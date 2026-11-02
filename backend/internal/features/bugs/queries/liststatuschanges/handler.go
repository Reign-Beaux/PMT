// Package liststatuschanges lista el historial de status de un bug.
package liststatuschanges

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	ListStatusChanges(ctx context.Context, id uuid.UUID) ([]domain.StatusChange, error)
}

// Item es un cambio de status.
type Item struct {
	ID           uuid.UUID     `json:"id"`
	BugID        uuid.UUID     `json:"bug_id"`
	FromStatus   domain.Status `json:"from_status"`
	ToStatus     domain.Status `json:"to_status"`
	ActorName    string        `json:"actor_name"`
	ActorProfile string        `json:"actor_profile"`
	Reason       string        `json:"reason"`
	ChangedAt    time.Time     `json:"changed_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el historial por ID, aunque el elemento ya no exista. Sin
// registros devuelve una lista vacía.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) ([]Item, error) {
	changes, err := h.reader.ListStatusChanges(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(changes))
	for _, change := range changes {
		items = append(items, Item{
			ID:           change.ID,
			BugID:        change.BugID,
			FromStatus:   change.From,
			ToStatus:     change.To,
			ActorName:    change.ActorName,
			ActorProfile: change.ActorProfile,
			Reason:       change.Reason,
			ChangedAt:    change.ChangedAt,
		})
	}
	return items, nil
}
