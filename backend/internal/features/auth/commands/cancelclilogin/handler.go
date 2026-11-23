// Package cancelclilogin cancela el código que pidió el CLI pmt: al sondear, el CLI sabe que no se autorizó.
package cancelclilogin

import (
	"context"
	"time"

	"pmt/internal/features/auth/domain"
	"pmt/internal/features/auth/queries/getclilogin"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.UserFinder
	CLILoginByCode(ctx context.Context, code string) (domain.CLILogin, error)
	CancelCLILogin(ctx context.Context, code string) (bool, error)
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

// Handle deja el código cancelado. Solo con el perfil User, y solo si sigue
// pendiente y vigente: si no, responde conflict.
func (h *Handler) Handle(ctx context.Context, cmd Command) (getclilogin.Item, error) {
	if _, err := domain.CurrentUser(ctx, h.repo, cmd.Actor); err != nil {
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
	done, err := h.repo.CancelCLILogin(ctx, login.Code)
	if err != nil {
		return getclilogin.Item{}, err
	}
	if !done {
		return getclilogin.Item{}, domain.ErrCLILoginClosed
	}
	login.State = domain.CLIStateCancelled
	return getclilogin.ItemOf(login, h.now()), nil
}
