// Package delete elimina un proyecto.
package delete

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
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

// Handle elimina el proyecto.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) error {
	p, err := h.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := p.EnsurePlanning(); err != nil {
		return err
	}
	return h.repo.Delete(ctx, id)
}
