// Package createtoken crea un token de API del usuario que actúa.
package createtoken

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	CreateAPIToken(ctx context.Context, id, userID uuid.UUID, name string, tokenHash []byte) (time.Time, error)
}

// Command contiene los datos de entrada.
type Command struct {
	Name  string       `json:"name"`
	Actor kernel.Actor `json:"-"`
}

// Result es el token creado. Token solo se muestra esta vez.
type Result struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Token     string    `json:"token"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle crea el token y guarda solo su hash.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	user, err := domain.CurrentUser(ctx, h.repo, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	name, err := domain.ValidateTokenName(cmd.Name)
	if err != nil {
		return Result{}, err
	}
	token, err := domain.NewAPIToken()
	if err != nil {
		return Result{}, err
	}
	id := uuid.New()
	created, err := h.repo.CreateAPIToken(ctx, id, user.ID, name, domain.HashToken(token))
	if err != nil {
		return Result{}, err
	}
	return Result{ID: id, Name: name, CreatedAt: created, Token: token}, nil
}
