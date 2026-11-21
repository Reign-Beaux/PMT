// Package disableuser desactiva a un usuario: no entra ni con cookie ni con token. Solo el dueño de la instancia.
package disableuser

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
	DisableUser(ctx context.Context, id uuid.UUID) (domain.User, error)
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

// Handle desactiva al usuario, cierra sus sesiones y revoca sus tokens. El dueño
// no se puede desactivar.
func (h *Handler) Handle(ctx context.Context, cmd Command) (domain.User, error) {
	if _, err := domain.CurrentOwner(ctx, h.repo, cmd.Actor); err != nil {
		return domain.User{}, err
	}
	target, err := h.repo.UserByID(ctx, cmd.ID)
	if err != nil {
		return domain.User{}, err
	}
	if target.IsInstanceOwner {
		return domain.User{}, domain.ErrOwnerCannotBeDisabled
	}
	return h.repo.DisableUser(ctx, target.ID)
}
