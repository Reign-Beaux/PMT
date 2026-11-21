// Package me devuelve el usuario que actúa.
package me

import (
	"context"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Handler ejecuta la operación.
type Handler struct {
	reader domain.UserFinder
}

// New crea el handler.
func New(reader domain.UserFinder) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el usuario del actor. Un agente no es usuario.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor) (domain.User, error) {
	return domain.CurrentUser(ctx, h.reader, actor)
}
