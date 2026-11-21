// Package acceptinvitation crea la cuenta del invitado y abre su sesión.
package acceptinvitation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	AcceptInvitation(ctx context.Context, tokenHash []byte, user domain.User, passwordHash string, sessionHash []byte, expires time.Time) (domain.User, error)
}

// Command contiene los datos de entrada. El correo es el de la invitación.
type Command struct {
	Token    string `json:"-"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Result es el usuario creado con el valor de su cookie de sesión.
type Result struct {
	User    domain.User
	Session string
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	agents domain.AgentNames
	now    func() time.Time
}

// New crea el handler.
func New(repo Repository, agents domain.AgentNames) *Handler {
	return &Handler{repo: repo, agents: agents, now: time.Now}
}

// Handle valida los datos, comprueba que el username no sea de un agente y crea
// la cuenta con una sesión de 30 días.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	name, err := domain.ValidateName(cmd.Name)
	if err != nil {
		return Result{}, err
	}
	username, err := domain.ValidateUsername(cmd.Username)
	if err != nil {
		return Result{}, err
	}
	if err = domain.ValidatePassword(cmd.Password); err != nil {
		return Result{}, err
	}
	if err = domain.EnsureUsernameFree(ctx, h.agents, username); err != nil {
		return Result{}, err
	}
	hash, err := domain.HashPassword(cmd.Password)
	if err != nil {
		return Result{}, err
	}
	session, err := domain.NewToken()
	if err != nil {
		return Result{}, err
	}
	user := domain.User{ID: uuid.New(), Username: username, Name: name}
	created, err := h.repo.AcceptInvitation(ctx, domain.HashToken(cmd.Token), user, hash, domain.HashToken(session), h.now().Add(domain.SessionTTL))
	if err != nil {
		return Result{}, err
	}
	return Result{User: created, Session: session}, nil
}
