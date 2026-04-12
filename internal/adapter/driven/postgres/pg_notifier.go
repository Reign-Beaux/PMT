package postgres

import (
	"context"
	"encoding/json"

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

func (n *PgNotifier) Notify(ownerID shared.ID, event notification.Event) {
	// Creamos un mapa plano para evitar problemas de tipos de Go.
	payloadBytes, _ := json.Marshal(event.Payload)
	
	msg := map[string]string{
		"owner_id":     ownerID.String(),
		"event_name":   event.Event,
		"payload_json": string(payloadBytes),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	sqlDB, err := n.db.DB()
	if err != nil {
		return
	}

	_, _ = sqlDB.ExecContext(context.Background(), "SELECT pg_notify($1, $2)", pgEventsChannel, string(data))
}
