// Package revoketoken revoca un token de API del usuario que actúa.
package revoketoken

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	RevokeAPIToken(ctx context.Context, id, userID uuid.UUID) error
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

// Handle revoca el token. Uno ajeno o ya revocado responde que no existe.
func (h *Handler) Handle(ctx context.Context, cmd Command) error {
	user, err := domain.CurrentUser(ctx, h.repo, cmd.Actor)
	if err != nil {
		return err
	}
	return h.repo.RevokeAPIToken(ctx, cmd.ID, user.ID)
}
