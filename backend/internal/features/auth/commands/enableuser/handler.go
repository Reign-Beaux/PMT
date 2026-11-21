// Package enableuser reactiva a un usuario desactivado. Solo el dueño de la instancia.
package enableuser

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	UserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	EnableUser(ctx context.Context, id uuid.UUID) (domain.User, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID    uuid.UUID
	Actor kernel.Actor
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle reactiva al usuario. Sus tokens revocados no vuelven: crea otros.
func (h *Handler) Handle(ctx context.Context, cmd Command) (domain.User, error) {
	if _, err := domain.CurrentOwner(ctx, h.repo, cmd.Actor); err != nil {
		return domain.User{}, err
	}
	target, err := h.repo.UserByID(ctx, cmd.ID)
	if err != nil {
		return domain.User{}, err
	}
	return h.repo.EnableUser(ctx, target.ID)
}
