package events

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Conn es la conexión dedicada que escucha el canal, fuera del pool de consultas.
type Conn interface {
	Listen(ctx context.Context) error
	WaitForNotification(ctx context.Context) (string, error)
	Close(ctx context.Context) error
}

// Dialer abre la conexión dedicada.
type Dialer interface {
	Dial(ctx context.Context) (Conn, error)
}

// PgxDialer abre conexiones pgx propias.
type PgxDialer struct {
	DatabaseURL string
}

// Dial conecta con pgx fuera del pool.
func (d PgxDialer) Dial(ctx context.Context) (Conn, error) {
	conn, err := pgx.Connect(ctx, d.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return pgxConn{conn: conn}, nil
}

type pgxConn struct {
	conn *pgx.Conn
}

func (c pgxConn) Listen(ctx context.Context) error {
	_, err := c.conn.Exec(ctx, "LISTEN "+Channel)
	return err
}

func (c pgxConn) WaitForNotification(ctx context.Context) (string, error) {
	notification, err := c.conn.WaitForNotification(ctx)
	if err != nil {
		return "", err
	}
	return notification.Payload, nil
}

func (c pgxConn) Close(ctx context.Context) error {
	return c.conn.Close(ctx)
}

const subscriberBuffer = 16

// Espera creciente entre reintentos; las pruebas la acortan.
var (
	initialRetryWait = time.Second
	maxRetryWait     = 30 * time.Second
)

// nextRetryWait duplica la espera sin pasar del tope.
func nextRetryWait(current time.Duration) time.Duration {
	if current <= 0 {
		return initialRetryWait
	}
	if next := current * 2; next < maxRetryWait {
		return next
	}
	return maxRetryWait
}

// Listener escucha el canal pmt_changes y reparte los avisos a suscriptores en
// memoria. Nunca devuelve error ni tumba el proceso: reintenta en segundo plano.
type Listener struct {
	dialer Dialer

	mu        sync.Mutex
	available bool
	stopped   bool
	subs      map[chan ChangeEvent]struct{}
}

// NewListener crea el Listener con su Dialer.
func NewListener(d Dialer) *Listener {
	return &Listener{dialer: d, subs: make(map[chan ChangeEvent]struct{})}
}

// Subscribe registra un suscriptor y devuelve su canal y su cancelación, que es
// idempotente. Con el listener disponible, el canal trae resync de inmediato; si
// Run ya terminó, devuelve un canal cerrado.
func (l *Listener) Subscribe() (<-chan ChangeEvent, func()) {
	ch := make(chan ChangeEvent, subscriberBuffer)

	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	l.subs[ch] = struct{}{}
	if l.available {
		ch <- ResyncEvent()
	}
	l.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			if _, ok := l.subs[ch]; ok {
				delete(l.subs, ch)
				close(ch)
			}
		})
	}
}

// Stopped dice si Run ya terminó; el handler WebSocket la usa para elegir el
// código de cierre.
func (l *Listener) Stopped() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopped
}

// Run bloquea hasta que ctx se cancela: conecta, escucha el canal y reparte los
// avisos. Al terminar cierra la conexión y todos los canales de suscriptores.
func (l *Listener) Run(ctx context.Context) {
	wait := time.Duration(0)
	for {
		if ctx.Err() != nil {
			l.shutdown()
			return
		}
		conn, err := l.dialer.Dial(ctx)
		if err != nil {
			log.Printf("events: conectar al canal %s: %v", Channel, err)
		} else if err := conn.Listen(ctx); err != nil {
			log.Printf("events: LISTEN %s: %v", Channel, err)
			l.closeConn(conn)
		} else {
			wait = 0
			l.markAvailable()
			if l.listen(ctx, conn) {
				return
			}
		}
		if ctx.Err() != nil {
			l.shutdown()
			return
		}
		l.markUnavailable()
		wait = nextRetryWait(wait)
		log.Printf("events: reintento en %s", wait)
		if !sleepCtx(ctx, wait) {
			l.shutdown()
			return
		}
	}
}

// listen reparte avisos hasta que la conexión falla o ctx se cancela. Devuelve
// true cuando el listener terminó por ctx.
func (l *Listener) listen(ctx context.Context, conn Conn) bool {
	for {
		payload, err := conn.WaitForNotification(ctx)
		if err != nil {
			l.closeConn(conn)
			if ctx.Err() != nil {
				l.shutdown()
				return true
			}
			log.Printf("events: esperar aviso de %s: %v", Channel, err)
			return false
		}
		l.handlePayload(payload)
	}
}

// handlePayload traduce el payload del NOTIFY y lo reparte. Uno inválido se
// registra y se descarta sin cortar el reparto.
func (l *Listener) handlePayload(payload string) {
	var message struct {
		Resource string `json:"resource"`
	}
	if err := json.Unmarshal([]byte(payload), &message); err != nil || !validResource(message.Resource) {
		log.Printf("events: payload inválido descartado: %q", payload)
		return
	}
	l.publish(ChangeEvent{Type: TypeInvalidate, Resource: message.Resource})
}

// publish entrega el evento a todos los suscriptores sin bloquearse; el que
// tenga el búfer lleno se descarta.
func (l *Listener) publish(event ChangeEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sendLocked(event)
}

// sendLocked reparte el evento; exige l.mu tomado.
func (l *Listener) sendLocked(event ChangeEvent) {
	for ch := range l.subs {
		select {
		case ch <- event:
		default:
			log.Printf("events: suscriptor lento descartado")
			delete(l.subs, ch)
			close(ch)
		}
	}
}

// markAvailable marca el listener disponible y envía resync a los suscriptores
// actuales, todo bajo el mismo cerrojo para que un Subscribe concurrente no
// reciba dos resync.
func (l *Listener) markAvailable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.available = true
	l.sendLocked(ResyncEvent())
}

func (l *Listener) markUnavailable() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.available = false
}

// shutdown marca el listener como terminado y cierra todos los canales.
func (l *Listener) shutdown() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopped = true
	l.available = false
	for ch := range l.subs {
		delete(l.subs, ch)
		close(ch)
	}
}

func (l *Listener) closeConn(conn Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Close(ctx); err != nil {
		log.Printf("events: cerrar conexión: %v", err)
	}
}

// sleepCtx espera d y devuelve false si ctx se canceló antes.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
