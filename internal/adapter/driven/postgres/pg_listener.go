package postgres

import (
	"context"
	"encoding/json"
	"fmt"
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
		// Connection errors are handled by the listener internally.
	})
	return &PgListener{listener: l, notifier: notifier}
}

func (l *PgListener) Start(ctx context.Context) error {
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
	var msg map[string]string
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return
	}

	ownerID, err := shared.ParseID(msg["owner_id"])
	if err != nil {
		return
	}

	var payload any
	if err := json.Unmarshal([]byte(msg["payload_json"]), &payload); err != nil {
		return
	}

	l.notifier.Notify(ownerID, notification.Event{
		Event:   msg["event_name"],
		Payload: payload,
	})
}
