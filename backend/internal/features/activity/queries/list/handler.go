// Package list consulta los comentarios de un elemento.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/activity/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, kind domain.ItemKind, itemID uuid.UUID) ([]domain.Comment, error)
}

// Query es la entrada de la operación.
type Query struct {
	ItemKind domain.ItemKind
	ItemID   uuid.UUID
}

// Item es un comentario de la lista.
type Item struct {
	ID           uuid.UUID       `json:"id"`
	ItemKind     domain.ItemKind `json:"item_kind"`
	ItemID       uuid.UUID       `json:"item_id"`
	Body         string          `json:"body"`
	ActorName    string          `json:"actor_name"`
	ActorProfile kernel.Profile  `json:"actor_profile"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los comentarios por fecha. No comprueba que el elemento exista,
// porque sus comentarios sobreviven a un elemento eliminado o retirado.
func (h *Handler) Handle(ctx context.Context, query Query) ([]Item, error) {
	if err := domain.EnsureKind(query.ItemKind); err != nil {
		return nil, err
	}
	comments, err := h.reader.List(ctx, query.ItemKind, query.ItemID)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(comments))
	for _, c := range comments {
		items = append(items, Item(c))
	}
	return items, nil
}
