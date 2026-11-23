// Package authorizeclilogin autoriza el código que pidió el CLI pmt: al sondear, el CLI recibe un token de API del usuario llamado "pmt <máquina>".
package authorizeclilogin

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
	"pmt/internal/features/auth/queries/getclilogin"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	CLILoginByCode(ctx context.Context, code string) (domain.CLILogin, error)
	AuthorizeCLILogin(ctx context.Context, code string, userID uuid.UUID) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	Code  string
	Actor kernel.Actor
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

// Handle deja el código autorizado. Solo con el perfil User, y solo si sigue
// pendiente y vigente: si no, responde conflict.
func (h *Handler) Handle(ctx context.Context, cmd Command) (getclilogin.Item, error) {
	user, err := domain.CurrentUser(ctx, h.repo, cmd.Actor)
	if err != nil {
		return getclilogin.Item{}, err
	}
	code, err := domain.NormalizeCLICode(cmd.Code)
	if err != nil {
		return getclilogin.Item{}, err
	}
	login, err := h.repo.CLILoginByCode(ctx, code)
	if err != nil {
		return getclilogin.Item{}, err
	}
	if login.Status(h.now()) != domain.CLIStatusPending {
		return getclilogin.Item{}, domain.ErrCLILoginClosed
	}
	done, err := h.repo.AuthorizeCLILogin(ctx, login.Code, user.ID)
	if err != nil {
		return getclilogin.Item{}, err
	}
	if !done {
		return getclilogin.Item{}, domain.ErrCLILoginClosed
	}
	login.State = domain.CLIStateAuthorized
	return getclilogin.ItemOf(login, h.now()), nil
}
