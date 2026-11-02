// Package revoke revoca a un agente: su token deja de valer.
package revoke

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/agents/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	GetByName(ctx context.Context, name string) (domain.Agent, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle revoca al agente. El historial conserva su nombre.
func (h *Handler) Handle(ctx context.Context, name string) error {
	agent, err := h.repo.GetByName(ctx, name)
	if err != nil {
		return err
	}
	return h.repo.Revoke(ctx, agent.ID)
}
