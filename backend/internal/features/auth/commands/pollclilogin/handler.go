// Package pollclilogin responde al CLI pmt en qué va su código. Es pública: el
// secreto de sondeo es la credencial. Con el código autorizado entrega el token
// una sola vez.
package pollclilogin

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	PollCLILogin(ctx context.Context, pollHash []byte, token domain.IssuedToken, now time.Time) (domain.CLIPoll, error)
}

// Command contiene los datos de entrada.
type Command struct {
	PollSecret string `json:"poll_secret"`
}

// Result es el estado del código: pending, authorized, cancelled o expired. Con
// authorized trae el token, que no se vuelve a mostrar.
type Result struct {
	Status   string     `json:"status"`
	Token    string     `json:"token,omitempty"`
	TokenID  *uuid.UUID `json:"token_id,omitempty"`
	Username string     `json:"username,omitempty"`
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

// Handle busca el código por el hash del secreto. El token se genera antes y
// solo se guarda si el código está autorizado.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	secret := strings.TrimSpace(cmd.PollSecret)
	if secret == "" {
		return Result{}, domain.ErrPollSecretRequired
	}
	token, err := domain.NewAPIToken()
	if err != nil {
		return Result{}, err
	}
	poll, err := h.repo.PollCLILogin(ctx, domain.HashToken(secret), domain.IssuedToken{ID: uuid.New(), Token: token}, h.now())
	if err != nil {
		return Result{}, err
	}
	if poll.Status != domain.CLIStatusAuthorized {
		return Result{Status: poll.Status}, nil
	}
	return Result{Status: poll.Status, Token: token, TokenID: &poll.TokenID, Username: poll.Username}, nil
}
