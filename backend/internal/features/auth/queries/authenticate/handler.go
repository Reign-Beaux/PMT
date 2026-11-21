// Package authenticate resuelve la sesión del navegador o un token de API de
// usuario a su actor.
package authenticate

import (
	"context"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación. Solo devuelve usuarios activos.
type Reader interface {
	UserBySession(ctx context.Context, sessionHash []byte) (domain.User, error)
	UserByAPIToken(ctx context.Context, tokenHash []byte) (domain.User, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Session devuelve el actor de una sesión vigente de un usuario activo, o
// kernel.ErrSessionInvalid.
func (h *Handler) Session(ctx context.Context, session string) (kernel.Actor, error) {
	if session == "" {
		return kernel.Actor{}, kernel.ErrSessionInvalid
	}
	user, err := h.reader.UserBySession(ctx, domain.HashToken(session))
	if err != nil {
		return kernel.Actor{}, err
	}
	return user.Actor(), nil
}

// Token devuelve el actor de un token de API vigente de un usuario activo, o
// kernel.ErrUnauthenticated.
func (h *Handler) Token(ctx context.Context, token string) (kernel.Actor, error) {
	if !domain.IsAPIToken(token) {
		return kernel.Actor{}, kernel.ErrUnauthenticated
	}
	user, err := h.reader.UserByAPIToken(ctx, domain.HashToken(token))
	if err != nil {
		return kernel.Actor{}, err
	}
	return user.Actor(), nil
}
