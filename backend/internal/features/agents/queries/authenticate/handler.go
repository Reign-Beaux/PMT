// Package authenticate resuelve un token a su actor.
package authenticate

import (
	"context"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	GetActiveByTokenHash(ctx context.Context, tokenHash []byte) (domain.Agent, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Authenticate devuelve el actor de un token activo, o kernel.ErrUnauthenticated.
// Implementa kernel.Authenticator.
func (h *Handler) Authenticate(ctx context.Context, token string) (kernel.Actor, error) {
	agent, err := h.reader.GetActiveByTokenHash(ctx, domain.HashToken(token))
	if err != nil {
		return kernel.Actor{}, err
	}
	return agent.Actor(), nil
}
