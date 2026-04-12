package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/gorm"

	"project-management-tools/internal/application/notification"
	"project-management-tools/internal/domain/shared"
)

const pgEventsChannel = "pmt_events"

type PgNotifier struct {
	db *gorm.DB
}

func NewPgNotifier(db *gorm.DB) *PgNotifier {
	return &PgNotifier{db: db}
}

func (n *PgNotifier) Notify(ctx context.Context, ownerID shared.ID, event notification.Event) {
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		log.Printf("pg_notifier: marshal payload: %v", err)
		return
	}

	msg := map[string]string{
		"owner_id":     ownerID.String(),
		"event_name":   event.Event,
		"payload_json": string(payloadBytes),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("pg_notifier: marshal message: %v", err)
		return
	}

	sqlDB, err := n.db.DB()
	if err != nil {
		log.Printf("pg_notifier: get sql.DB: %v", err)
		return
	}

	if _, err := sqlDB.ExecContext(ctx, "SELECT pg_notify($1, $2)", pgEventsChannel, string(data)); err != nil {
		log.Printf("pg_notifier: pg_notify: %v", err)
		return
	}

	log.Printf("pg_notifier: sent %s to %s", event.Event, fmt.Sprintf("%.8s", ownerID.String()))
}
