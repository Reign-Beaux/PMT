// Package createinvitation invita a un correo. Solo el dueño de la instancia.
package createinvitation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	UserByEmail(ctx context.Context, email string) (domain.User, string, error)
	CreateInvitation(ctx context.Context, id uuid.UUID, email string, tokenHash []byte, invitedBy uuid.UUID, expires time.Time) (time.Time, error)
}

// Command contiene los datos de entrada.
type Command struct {
	Email string       `json:"email"`
	Actor kernel.Actor `json:"-"`
}

// Result es la invitación con el enlace que el dueño copia y manda. El enlace no
// se puede volver a obtener.
type Result struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo      Repository
	publicURL string
	now       func() time.Time
}

// New crea el handler. publicURL es app.public_url, la base del enlace.
func New(repo Repository, publicURL string) *Handler {
	return &Handler{repo: repo, publicURL: publicURL, now: time.Now}
}

// Handle crea una invitación de 7 días a un correo sin usuario. No envía correos.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	owner, err := domain.CurrentOwner(ctx, h.repo, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	email, err := domain.ValidateEmail(cmd.Email)
	if err != nil {
		return Result{}, err
	}
	_, _, err = h.repo.UserByEmail(ctx, email)
	if err == nil {
		return Result{}, domain.ErrEmailTaken
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return Result{}, err
	}
	token, err := domain.NewToken()
	if err != nil {
		return Result{}, err
	}
	id := uuid.New()
	expires, err := h.repo.CreateInvitation(ctx, id, email, domain.HashToken(token), owner.ID, h.now().Add(domain.InvitationTTL))
	if err != nil {
		return Result{}, err
	}
	return Result{ID: id, Email: email, URL: domain.InvitationURL(h.publicURL, token), ExpiresAt: expires}, nil
}
