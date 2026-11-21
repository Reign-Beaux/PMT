// Package logout cierra la sesión del navegador.
package logout

import (
	"context"

	"pmt/internal/features/auth/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	DeleteSession(ctx context.Context, sessionHash []byte) error
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle borra la sesión. Sin sesión no hace nada.
func (h *Handler) Handle(ctx context.Context, session string) error {
	if session == "" {
		return nil
	}
	return h.repo.DeleteSession(ctx, domain.HashToken(session))
}
