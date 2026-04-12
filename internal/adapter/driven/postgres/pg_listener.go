package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"

	"project-management-tools/internal/application/notification"
	"project-management-tools/internal/domain/shared"
)

type PgListener struct {
	listener *pq.Listener
	notifier notification.Notifier
}

func NewPgListener(dsn string, notifier notification.Notifier) *PgListener {
	l := pq.NewListener(dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("PgListener: connection event: %v", err)
		}
	})
	return &PgListener{listener: l, notifier: notifier}
}

func (l *PgListener) Start(ctx context.Context) error {
	log.Printf("PgListener: starting subscription to %s", pgEventsChannel)
	if err := l.listener.Listen(pgEventsChannel); err != nil {
		return fmt.Errorf("PgListener: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return l.listener.Close()
		case n, ok := <-l.listener.Notify:
			if !ok {
				return nil
			}
			if n != nil {
				l.dispatch(n.Extra)
			}
		}
	}
}

func (l *PgListener) dispatch(raw string) {
	// Intentamos decodificar en un mapa genérico para ser flexibles
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		log.Printf("PgListener: Error decoding raw JSON: %v", err)
		return
	}

	ownerIDStr, _ := data["owner_id"].(string)
	if ownerIDStr == "" {
		// Intentamos con el nombre corto si existe
		ownerIDStr, _ = data["oid"].(string)
	}
	
	ownerID, _ := shared.ParseID(ownerIDStr)

	// Extraemos el evento
	var eventName string
	var payload any

	// CASO A: El evento viene anidado en una propiedad "event" (formato viejo)
	if e, ok := data["event"].(map[string]any); ok {
		eventName, _ = e["event"].(string)
		payload = e["payload"]
	} else if e, ok := data["event_data"].(map[string]any); ok {
		// CASO B: Formato intermedio "event_data"
		eventName, _ = e["event"].(string)
		payload = e["payload"]
	} else {
		// CASO C: El evento viene plano (formato nuevo)
		eventName, _ = data["event_name"].(string)
		if pStr, ok := data["payload_json"].(string); ok {
			_ = json.Unmarshal([]byte(pStr), &payload)
		}
	}

	if eventName == "" {
		log.Printf("PgListener: Could not find event name in data: %s", raw)
		return
	}

	log.Printf("PgListener: Forwarding event '%s' to Hub", eventName)
	l.notifier.Notify(ownerID, notification.Event{
		Event:   eventName,
		Payload: payload,
	})
}
