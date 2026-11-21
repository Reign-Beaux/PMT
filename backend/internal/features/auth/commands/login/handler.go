// Package login abre una sesión con correo y contraseña.
package login

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// ErrBadCredentials no dice si falló el correo o la contraseña.
var ErrBadCredentials = kernel.Unauthorized("correo o contraseña incorrectos")

// ErrDisabled indica que el usuario está desactivado.
var ErrDisabled = kernel.Forbidden("tu usuario está desactivado")

// Repository es la persistencia que necesita la operación.
type Repository interface {
	UserByEmail(ctx context.Context, email string) (domain.User, string, error)
	CreateSession(ctx context.Context, sessionHash []byte, userID uuid.UUID, expires time.Time) error
}

// Command contiene los datos de entrada.
type Command struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Result es el usuario con el valor de su cookie de sesión.
type Result struct {
	User    domain.User
	Session string
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
	now  func() time.Time
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo, now: time.Now}
}

// Handle verifica la contraseña y abre una sesión de 30 días.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if cmd.Password == "" {
		return Result{}, ErrBadCredentials
	}
	user, hash, err := h.repo.UserByEmail(ctx, domain.NormalizeEmail(cmd.Email))
	if errors.Is(err, domain.ErrUserNotFound) {
		return Result{}, ErrBadCredentials
	}
	if err != nil {
		return Result{}, err
	}
	ok, err := domain.VerifyPassword(cmd.Password, hash)
	if err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{}, ErrBadCredentials
	}
	if user.Disabled {
		return Result{}, ErrDisabled
	}
	session, err := domain.NewToken()
	if err != nil {
		return Result{}, err
	}
	if err = h.repo.CreateSession(ctx, domain.HashToken(session), user.ID, h.now().Add(domain.SessionTTL)); err != nil {
		return Result{}, err
	}
	return Result{User: user, Session: session}, nil
}
