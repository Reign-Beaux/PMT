package events

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// wsTest monta un servidor httptest con el handler y un listener alimentado por
// el fakeDialer.
type wsTest struct {
	ts     *httptest.Server
	l      *Listener
	dialer *fakeDialer
	cancel context.CancelFunc
	done   chan struct{}
}

func startWS(t *testing.T) *wsTest {
	t.Helper()
	original := initialRetryWait
	initialRetryWait = 10 * time.Millisecond

	dialer := newFakeDialer()
	l := NewListener(dialer)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		l.Run(ctx)
		close(done)
	}()
	ts := httptest.NewServer(Handler(l))

	t.Cleanup(func() {
		cancel()
		<-done
		ts.Close()
		initialRetryWait = original
	})
	return &wsTest{ts: ts, l: l, dialer: dialer, cancel: cancel, done: done}
}

// connect acepta la conexión del listener y abre un cliente WebSocket.
func (w *wsTest) connect(t *testing.T, conn *fakeConn) *websocket.Conn {
	t.Helper()
	w.dialer.push(conn)
	sub, cancel := w.l.Subscribe()
	defer cancel()
	waitEvent(t, sub) // resync: el listener ya está disponible

	ctx, cancelDial := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDial()
	url := "ws" + strings.TrimPrefix(w.ts.URL, "http")
	client, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() { client.CloseNow() })
	return client
}

func readFrame(t *testing.T, client *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	typ, data, err := client.Read(ctx)
	if err != nil {
		t.Fatalf("leer frame: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("tipo de mensaje = %v, quiero texto", typ)
	}
	return string(data)
}

func TestWSPrimerFrameResyncEInvalidate(t *testing.T) {
	w := startWS(t)
	conn := newFakeConn()
	client := w.connect(t, conn)

	if frame := readFrame(t, client); frame != `{"type":"resync"}` {
		t.Fatalf("primer frame = %s, quiero {\"type\":\"resync\"}", frame)
	}

	conn.payloads <- `{"resource":"task"}`
	if frame := readFrame(t, client); frame != `{"type":"invalidate","resource":"task"}` {
		t.Fatalf("frame = %s, quiero invalidate task", frame)
	}
}

func TestWSResyncTrasReconexion(t *testing.T) {
	w := startWS(t)
	first := newFakeConn()
	client := w.connect(t, first)
	readFrame(t, client) // resync inicial

	first.drop()
	second := newFakeConn()
	w.dialer.push(second)

	if frame := readFrame(t, client); frame != `{"type":"resync"}` {
		t.Fatalf("frame = %s, quiero resync tras reconexión", frame)
	}

	second.payloads <- `{"resource":"project"}`
	if frame := readFrame(t, client); frame != `{"type":"invalidate","resource":"project"}` {
		t.Fatalf("frame = %s, quiero invalidate project", frame)
	}
}

func TestWSDosClientesRecibenElMismoEvento(t *testing.T) {
	w := startWS(t)
	conn := newFakeConn()
	client1 := w.connect(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(w.ts.URL, "http")
	client2, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial segundo cliente: %v", err)
	}
	t.Cleanup(func() { client2.CloseNow() })

	readFrame(t, client1) // resync
	if frame := readFrame(t, client2); frame != `{"type":"resync"}` {
		t.Fatalf("cliente2 primer frame = %s, quiero resync", frame)
	}

	conn.payloads <- `{"resource":"bug"}`
	want := `{"type":"invalidate","resource":"bug"}`
	if frame := readFrame(t, client1); frame != want {
		t.Fatalf("cliente1 frame = %s, quiero %s", frame, want)
	}
	if frame := readFrame(t, client2); frame != want {
		t.Fatalf("cliente2 frame = %s, quiero %s", frame, want)
	}
}

func TestWSMensajeDelClienteNoCierraLaConexion(t *testing.T) {
	w := startWS(t)
	conn := newFakeConn()
	client := w.connect(t, conn)
	readFrame(t, client) // resync

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Write(ctx, websocket.MessageText, []byte("hola")); err != nil {
		t.Fatalf("escribir al servidor: %v", err)
	}

	conn.payloads <- `{"resource":"comment"}`
	if frame := readFrame(t, client); frame != `{"type":"invalidate","resource":"comment"}` {
		t.Fatalf("frame = %s, quiero invalidate comment", frame)
	}
}

func TestWSCierreDelServidorTerminaElHandler(t *testing.T) {
	w := startWS(t)
	conn := newFakeConn()
	client := w.connect(t, conn)
	readFrame(t, client) // resync

	w.cancel()
	select {
	case <-w.done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run no devolvió al cancelar ctx")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := client.Read(ctx)
	if err == nil {
		t.Fatal("el cliente no recibió el cierre")
	}
	if code := websocket.CloseStatus(err); code != websocket.StatusGoingAway {
		t.Fatalf("código de cierre = %v, quiero GoingAway", code)
	}

	closed := make(chan struct{})
	go func() {
		w.ts.Close() // espera a que los handlers activos terminen
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("el handler no terminó: ts.Close no devuelve")
	}
}

func TestWSPeticionSinUpgrade(t *testing.T) {
	w := startWS(t)
	resp, err := http.Get(w.ts.URL + "/api/events")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSwitchingProtocols {
		t.Fatal("una petición sin upgrade no debe recibir 101")
	}
	// La biblioteca responde 426 Upgrade Required sin cabeceras de WebSocket.
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("status = %d, quiero 426", resp.StatusCode)
	}
}
