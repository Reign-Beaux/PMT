package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeConn es una conexión dedicada falsa alimentada por la prueba.
type fakeConn struct {
	payloads  chan string
	fail      chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		payloads: make(chan string),
		fail:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}

func (c *fakeConn) Listen(context.Context) error { return nil }

func (c *fakeConn) WaitForNotification(ctx context.Context) (string, error) {
	select {
	case payload := <-c.payloads:
		return payload, nil
	case <-c.fail:
		return "", errors.New("fake: conexión perdida")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *fakeConn) Close(context.Context) error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

// drop simula la pérdida de la conexión.
func (c *fakeConn) drop() { close(c.fail) }

type dialResult struct {
	conn Conn
	err  error
}

// fakeDialer entrega conexiones falsas cuando la prueba las empuja.
type fakeDialer struct {
	results chan dialResult
	dialed  chan struct{}
}

func newFakeDialer() *fakeDialer {
	return &fakeDialer{
		results: make(chan dialResult, 8),
		dialed:  make(chan struct{}, 64),
	}
}

func (d *fakeDialer) Dial(ctx context.Context) (Conn, error) {
	select {
	case d.dialed <- struct{}{}:
	default:
	}
	select {
	case result := <-d.results:
		return result.conn, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (d *fakeDialer) push(conn Conn) { d.results <- dialResult{conn: conn} }

func (d *fakeDialer) pushErr(err error) { d.results <- dialResult{err: err} }

// runningListener arranca Run en segundo plano; la limpieza lo detiene y espera.
type runningListener struct {
	*Listener
	cancel context.CancelFunc
	done   chan struct{}
}

func startListener(t *testing.T, d Dialer) *runningListener {
	t.Helper()
	l := NewListener(d)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		l.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return &runningListener{Listener: l, cancel: cancel, done: done}
}

func waitEvent(t *testing.T, ch <-chan ChangeEvent) ChangeEvent {
	t.Helper()
	select {
	case event, ok := <-ch:
		if !ok {
			t.Fatal("canal cerrado inesperadamente")
		}
		return event
	case <-time.After(5 * time.Second):
		t.Fatal("espera de evento agotada")
		return ChangeEvent{}
	}
}

func expectNoEvent(t *testing.T, ch <-chan ChangeEvent, d time.Duration) {
	t.Helper()
	select {
	case event, ok := <-ch:
		t.Fatalf("evento inesperado %+v (ok=%v)", event, ok)
	case <-time.After(d):
	}
}

func expectClosed(t *testing.T, ch <-chan ChangeEvent) {
	t.Helper()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-time.After(5 * time.Second):
			t.Fatal("el canal no se cerró")
		}
	}
}

func TestRepartoAVariosSuscriptores(t *testing.T) {
	dialer := newFakeDialer()
	conn := newFakeConn()
	dialer.push(conn)
	running := startListener(t, dialer)

	sub1, cancel1 := running.Subscribe()
	defer cancel1()
	sub2, cancel2 := running.Subscribe()
	defer cancel2()

	if event := waitEvent(t, sub1); event != (ChangeEvent{Type: TypeResync}) {
		t.Fatalf("sub1: primer evento = %+v, quiero resync", event)
	}
	if event := waitEvent(t, sub2); event != (ChangeEvent{Type: TypeResync}) {
		t.Fatalf("sub2: primer evento = %+v, quiero resync", event)
	}

	conn.payloads <- `{"resource":"task"}`
	want := ChangeEvent{Type: TypeInvalidate, Resource: ResourceTask}
	if event := waitEvent(t, sub1); event != want {
		t.Fatalf("sub1: evento = %+v, quiero %+v", event, want)
	}
	if event := waitEvent(t, sub2); event != want {
		t.Fatalf("sub2: evento = %+v, quiero %+v", event, want)
	}
}

func TestPayloadInvalidoSeDescarta(t *testing.T) {
	dialer := newFakeDialer()
	conn := newFakeConn()
	dialer.push(conn)
	running := startListener(t, dialer)

	sub, cancel := running.Subscribe()
	defer cancel()
	waitEvent(t, sub) // resync inicial

	for _, invalid := range []string{
		`no es json`,
		`{"resource":"recurso_inexistente"}`,
		`{"resource":""}`,
		`{"otro":"campo"}`,
		`{"resource":"project"`,
	} {
		conn.payloads <- invalid
	}
	conn.payloads <- `{"resource" : "project"}`

	if event := waitEvent(t, sub); event != (ChangeEvent{Type: TypeInvalidate, Resource: ResourceProject}) {
		t.Fatalf("evento = %+v, quiero invalidate project", event)
	}
}

func TestResyncSoloConListenerDisponible(t *testing.T) {
	dialer := newFakeDialer()
	running := startListener(t, dialer)

	sub, cancel := running.Subscribe()
	defer cancel()
	expectNoEvent(t, sub, 150*time.Millisecond) // aún no hay conexión

	conn := newFakeConn()
	dialer.push(conn)
	if event := waitEvent(t, sub); event != (ChangeEvent{Type: TypeResync}) {
		t.Fatalf("evento = %+v, quiero resync al conectar", event)
	}

	late, cancelLate := running.Subscribe()
	defer cancelLate()
	if event := waitEvent(t, late); event != (ChangeEvent{Type: TypeResync}) {
		t.Fatalf("suscriptor con listener disponible: evento = %+v, quiero resync", event)
	}
}

func TestReintentaConBaseCaidaAlArrancar(t *testing.T) {
	original := initialRetryWait
	initialRetryWait = 10 * time.Millisecond
	defer func() { initialRetryWait = original }()

	dialer := newFakeDialer()
	running := startListener(t, dialer)

	sub, cancel := running.Subscribe()
	defer cancel()

	dialer.pushErr(errors.New("fake: base caída"))
	dialer.pushErr(errors.New("fake: base caída"))

	conn := newFakeConn()
	dialer.push(conn)
	if event := waitEvent(t, sub); event != (ChangeEvent{Type: TypeResync}) {
		t.Fatalf("evento = %+v, quiero resync al recuperarse", event)
	}
	if dialed := len(dialer.dialed); dialed < 3 {
		t.Fatalf("intentos de conexión = %d, quiero al menos 3", dialed)
	}
}

func TestResyncTrasReconexion(t *testing.T) {
	original := initialRetryWait
	initialRetryWait = 10 * time.Millisecond
	defer func() { initialRetryWait = original }()

	dialer := newFakeDialer()
	first := newFakeConn()
	dialer.push(first)
	running := startListener(t, dialer)

	sub, cancel := running.Subscribe()
	defer cancel()
	waitEvent(t, sub) // resync inicial

	first.drop()
	second := newFakeConn()
	dialer.push(second)

	if event := waitEvent(t, sub); event != (ChangeEvent{Type: TypeResync}) {
		t.Fatalf("evento = %+v, quiero resync tras reconexión", event)
	}

	second.payloads <- `{"resource":"bug"}`
	if event := waitEvent(t, sub); event != (ChangeEvent{Type: TypeInvalidate, Resource: ResourceBug}) {
		t.Fatalf("evento = %+v, quiero invalidate bug", event)
	}
	select {
	case <-first.closed:
	default:
		t.Error("la conexión perdida no se cerró")
	}
}

func TestEsperaCrecienteConTope(t *testing.T) {
	want := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second,
		30 * time.Second,
	}
	current := time.Duration(0)
	for i, expected := range want {
		current = nextRetryWait(current)
		if current != expected {
			t.Fatalf("espera %d = %s, quiero %s", i+1, current, expected)
		}
	}
	if maxRetryWait != 30*time.Second {
		t.Fatalf("tope = %s, quiero 30s", maxRetryWait)
	}
}

func TestSuscriptorLentoDescartado(t *testing.T) {
	dialer := newFakeDialer()
	conn := newFakeConn()
	dialer.push(conn)
	running := startListener(t, dialer)

	slow, cancelSlow := running.Subscribe()
	defer cancelSlow()
	fast, cancelFast := running.Subscribe()
	defer cancelFast()

	waitEvent(t, slow) // resync; a partir de aquí no se lee más
	waitEvent(t, fast)

	comment := ChangeEvent{Type: TypeInvalidate, Resource: ResourceComment}
	total := subscriberBuffer + 1
	for i := 0; i < total; i++ {
		conn.payloads <- `{"resource":"comment"}`
		// El rápido se lee en cada paso para que nunca se llene.
		if event := waitEvent(t, fast); event != comment {
			t.Fatalf("rápido: evento %d = %+v, quiero %+v", i+1, event, comment)
		}
	}

	// El lento se llenó y fue descartado: su canal queda cerrado tras vaciarse.
	expectClosed(t, slow)

	conn.payloads <- `{"resource":"stage"}`
	stage := ChangeEvent{Type: TypeInvalidate, Resource: ResourceStage}
	if event := waitEvent(t, fast); event != stage {
		t.Fatalf("rápido: evento final = %+v, quiero %+v", event, stage)
	}
}

func TestCierreLimpio(t *testing.T) {
	dialer := newFakeDialer()
	conn := newFakeConn()
	dialer.push(conn)
	running := startListener(t, dialer)

	sub1, cancel1 := running.Subscribe()
	sub2, cancel2 := running.Subscribe()
	waitEvent(t, sub1)
	waitEvent(t, sub2)

	running.cancel()
	select {
	case <-running.done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run no devolvió al cancelar ctx")
	}

	expectClosed(t, sub1)
	expectClosed(t, sub2)
	select {
	case <-conn.closed:
	default:
		t.Error("la conexión no se cerró")
	}

	cancel1()
	cancel1() // idempotente
	cancel2()
	cancel1() // después del cierre, sin pánico

	late, cancelLate := running.Subscribe()
	defer cancelLate()
	if _, ok := <-late; ok {
		t.Fatal("Subscribe tras Run terminado debe devolver un canal cerrado")
	}
}
