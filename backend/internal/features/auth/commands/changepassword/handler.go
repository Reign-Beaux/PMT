// Package changepassword cambia la contraseña del usuario que actúa.
package changepassword

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

var (
	// ErrSessionRequired indica que se intentó sin la sesión del navegador.
	ErrSessionRequired = kernel.Unauthorized("se necesita la sesión del navegador")
	// ErrCurrentPasswordWrong indica que la contraseña actual no coincide.
	ErrCurrentPasswordWrong = kernel.Invalid("la contraseña actual no coincide")
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	PasswordHash(ctx context.Context, id uuid.UUID) (string, error)
	ChangePassword(ctx context.Context, id uuid.UUID, passwordHash string, keepSession []byte) error
}

// Command contiene los datos de entrada.
type Command struct {
	Current string       `json:"current_password"`
	New     string       `json:"new_password"`
	Session string       `json:"-"`
	Actor   kernel.Actor `json:"-"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle verifica la contraseña actual, guarda la nueva y cierra las demás
// sesiones. Solo desde el navegador: con un token no hay sesión que conservar.
func (h *Handler) Handle(ctx context.Context, cmd Command) error {
	if cmd.Session == "" {
		return ErrSessionRequired
	}
	user, err := domain.CurrentUser(ctx, h.repo, cmd.Actor)
	if err != nil {
		return err
	}
	if err = domain.ValidatePassword(cmd.New); err != nil {
		return err
	}
	hash, err := h.repo.PasswordHash(ctx, user.ID)
	if err != nil {
		return err
	}
	ok, err := domain.VerifyPassword(cmd.Current, hash)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCurrentPasswordWrong
	}
	hash, err = domain.HashPassword(cmd.New)
	if err != nil {
		return err
	}
	return h.repo.ChangePassword(ctx, user.ID, hash, domain.HashToken(cmd.Session))
}
