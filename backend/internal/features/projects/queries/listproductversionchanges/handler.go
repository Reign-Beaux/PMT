// Package listproductversionchanges lista el historial de versiones de producto de un proyecto.
package listproductversionchanges

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	ListProductVersionChanges(ctx context.Context, id uuid.UUID) ([]domain.ProductVersionChange, error)
}

// Item es un cambio de versión de producto.
type Item struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    uuid.UUID `json:"project_id"`
	FromVersion  string    `json:"from_version"`
	ToVersion    string    `json:"to_version"`
	ActorName    string    `json:"actor_name"`
	ActorProfile string    `json:"actor_profile"`
	Reason       string    `json:"reason"`
	ChangedAt    time.Time `json:"changed_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el historial por ID en orden cronológico, aunque el proyecto ya
// no exista. Sin registros devuelve una lista vacía.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) ([]Item, error) {
	changes, err := h.reader.ListProductVersionChanges(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(changes))
	for _, change := range changes {
		items = append(items, Item{
			ID:           change.ID,
			ProjectID:    change.ProjectID,
			FromVersion:  change.From,
			ToVersion:    change.To,
			ActorName:    change.ActorName,
			ActorProfile: change.ActorProfile,
			Reason:       change.Reason,
			ChangedAt:    change.ChangedAt,
		})
	}
	return items, nil
}
