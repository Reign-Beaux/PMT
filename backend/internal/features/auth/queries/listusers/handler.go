// Package listusers lista los usuarios. Solo el dueño de la instancia.
package listusers

import (
	"context"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	domain.UserFinder
	ListUsers(ctx context.Context) ([]domain.User, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los usuarios por username, activos y desactivados.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor) ([]domain.User, error) {
	if _, err := domain.CurrentOwner(ctx, h.reader, actor); err != nil {
		return nil, err
	}
	return h.reader.ListUsers(ctx)
}
