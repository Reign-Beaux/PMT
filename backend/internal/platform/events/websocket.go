package events

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// Intervalos del handler; las pruebas los acortan.
var (
	pingInterval = 30 * time.Second
	writeTimeout = 5 * time.Second
	pingTimeout  = 10 * time.Second
)

// Handler devuelve el handler de GET /api/events: acepta la conexión WebSocket
// y escribe al cliente los ChangeEvent del listener como frames de texto JSON.
// Los mensajes del cliente se ignoran.
func Handler(l *Listener) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// InsecureSkipVerify: el proxy de Vite usa changeOrigin y reescribe el
		// Host, y el endpoint solo emite nombres de recurso.
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			log.Printf("events: aceptar websocket: %v", err)
			return
		}
		defer conn.CloseNow()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		events, unsubscribe := l.Subscribe()
		defer unsubscribe()

		// Lectora: descarta los mensajes del cliente; si la lectura falla
		// (cliente muerto o cortado), termina el handler.
		go func() {
			for {
				if _, _, err := conn.Read(ctx); err != nil {
					cancel()
					return
				}
			}
		}()

		ping := time.NewTicker(pingInterval)
		defer ping.Stop()

		for {
			select {
			case <-ctx.Done():
				// El cliente se desconectó o la petición terminó; el CloseNow
				// del defer libera la conexión.
				return
			case event, ok := <-events:
				if !ok {
					code, reason := websocket.StatusTryAgainLater, "suscripción descartada por lenta"
					if l.Stopped() {
						code, reason = websocket.StatusGoingAway, "servidor cerrado"
					}
					closeWS(conn, code, reason)
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					log.Printf("events: serializar %+v: %v", event, err)
					continue
				}
				writeCtx, cancelWrite := context.WithTimeout(ctx, writeTimeout)
				err = conn.Write(writeCtx, websocket.MessageText, data)
				cancelWrite()
				if err != nil {
					log.Printf("events: escribir al cliente: %v", err)
					closeWS(conn, websocket.StatusGoingAway, "error de escritura")
					return
				}
			case <-ping.C:
				pingCtx, cancelPing := context.WithTimeout(ctx, pingTimeout)
				err := conn.Ping(pingCtx)
				cancelPing()
				if err != nil {
					log.Printf("events: ping al cliente: %v", err)
					closeWS(conn, websocket.StatusGoingAway, "ping fallido")
					return
				}
			}
		}
	})
}

// closeWS cierra con un código explícito para que el cliente sepa que debe
// reconectar; si el frame de cierre no llega, CloseNow del defer libera.
func closeWS(conn *websocket.Conn, code websocket.StatusCode, reason string) {
	if err := conn.Close(code, reason); err != nil {
		log.Printf("events: cerrar websocket (%d %s): %v", code, reason, err)
	}
}
