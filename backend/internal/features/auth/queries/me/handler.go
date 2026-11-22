// Package me devuelve el usuario que actúa y el perfil con el que actúa.
package me

import (
	"context"
	"errors"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Result es el usuario y el perfil efectivo de la petición: User con la cookie
// o sin perfil declarado; el declarado, con un token de usuario.
type Result struct {
	domain.User
	Profile kernel.Profile `json:"profile"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader domain.UserFinder
}

// New crea el handler.
func New(reader domain.UserFinder) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el usuario del actor con cualquier perfil. Un agente no es
// usuario: su nombre nunca coincide con un username.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor) (Result, error) {
	if actor.Name == "" {
		return Result{}, domain.ErrUsersOnly
	}
	user, err := h.reader.UserByUsername(ctx, actor.Name)
	if errors.Is(err, domain.ErrUserNotFound) {
		return Result{}, domain.ErrUsersOnly
	}
	if err != nil {
		return Result{}, err
	}
	return Result{User: user, Profile: actor.Profile}, nil
}
