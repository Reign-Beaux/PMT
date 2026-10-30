// Package delete elimina una tarea.
package delete

import (
	"context"

	"github.com/google/uuid"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Delete(ctx context.Context, id uuid.UUID) error
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle elimina la tarea.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) error {
	return h.repo.Delete(ctx, id)
}
