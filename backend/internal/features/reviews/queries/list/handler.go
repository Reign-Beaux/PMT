// Package list consulta los hallazgos de una tarea o un bug.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, kind domain.ItemKind, itemID uuid.UUID, pending bool) ([]domain.Finding, error)
}

// Item es un hallazgo de la lista.
type Item struct {
	ID           uuid.UUID       `json:"id"`
	ItemKind     domain.ItemKind `json:"item_kind"`
	ItemID       uuid.UUID       `json:"item_id"`
	Description  string          `json:"description"`
	Blocking     bool            `json:"blocking"`
	Status       domain.Status   `json:"status"`
	ActorName    string          `json:"actor_name"`
	ActorProfile kernel.Profile  `json:"actor_profile"`
	ClosedBy     string          `json:"closed_by,omitempty"`
	CloseReason  string          `json:"close_reason,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	ClosedAt     *time.Time      `json:"closed_at,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los hallazgos. No comprueba que el elemento exista: sobreviven a
// él, y uno sin hallazgos responde lista vacía.
func (h *Handler) Handle(ctx context.Context, kind domain.ItemKind, itemID uuid.UUID, pending bool) ([]Item, error) {
	if err := domain.EnsureKind(kind); err != nil {
		return nil, err
	}
	findings, err := h.reader.List(ctx, kind, itemID, pending)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(findings))
	for _, f := range findings {
		items = append(items, Item(f))
	}
	return items, nil
}
