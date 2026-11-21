// Package listtokens lista los tokens vigentes del usuario que actúa.
package listtokens

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	domain.UserFinder
	ListAPITokens(ctx context.Context, userID uuid.UUID) ([]domain.APIToken, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los tokens sin revocar, sin el token en sí.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor) ([]domain.APIToken, error) {
	user, err := domain.CurrentUser(ctx, h.reader, actor)
	if err != nil {
		return nil, err
	}
	return h.reader.ListAPITokens(ctx, user.ID)
}
