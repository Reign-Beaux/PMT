// Package listinvitations lista las invitaciones. Solo el dueño de la instancia.
package listinvitations

import (
	"context"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	domain.UserFinder
	ListInvitations(ctx context.Context) ([]domain.Invitation, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve las invitaciones, de la más nueva a la más vieja.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor) ([]domain.Invitation, error) {
	if _, err := domain.CurrentOwner(ctx, h.reader, actor); err != nil {
		return nil, err
	}
	return h.reader.ListInvitations(ctx)
}
