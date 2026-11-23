// Package startclilogin da al CLI pmt un código para iniciar sesión por el
// navegador. Es pública: quien lo pide todavía no tiene credenciales.
package startclilogin

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	StartCLILogin(ctx context.Context, login domain.NewCLILogin, maxPending int) error
}

// Command contiene los datos de entrada.
type Command struct {
	// Machine es el nombre de la máquina: el token se llamará "pmt <máquina>".
	Machine string `json:"machine"`
}

// Result es lo que el CLI necesita. PollSecret solo lo conoce el CLI; PMT guarda su hash.
type Result struct {
	Code       string    `json:"code"`
	PollSecret string    `json:"poll_secret"`
	URL        string    `json:"url"`
	ExpiresAt  time.Time `json:"expires_at"`
	Interval   int       `json:"interval"`
}

// codeAttempts es cuántas veces se genera otro código si sale uno repetido.
const codeAttempts = 5

// Handler ejecuta la operación.
type Handler struct {
	repo      Repository
	publicURL string
	now       func() time.Time
}

// New crea el handler. publicURL es app.public_url, la base de la pantalla /cli.
func New(repo Repository, publicURL string) *Handler {
	return &Handler{repo: repo, publicURL: publicURL, now: time.Now}
}

// Handle genera el código y el secreto de sondeo, y guarda solo el hash del
// secreto. Responde conflict si ya hay demasiados códigos pendientes.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	machine, err := domain.ValidateMachine(cmd.Machine)
	if err != nil {
		return Result{}, err
	}
	secret, err := domain.NewToken()
	if err != nil {
		return Result{}, err
	}
	expires := h.now().Add(domain.CLILoginTTL)
	for range codeAttempts {
		code, err := domain.NewCLICode()
		if err != nil {
			return Result{}, err
		}
		err = h.repo.StartCLILogin(ctx, domain.NewCLILogin{
			ID: uuid.New(), Code: code, PollHash: domain.HashToken(secret), Machine: machine, ExpiresAt: expires,
		}, domain.MaxPendingCLILogins)
		if errors.Is(err, domain.ErrCLICodeTaken) {
			continue
		}
		if err != nil {
			return Result{}, err
		}
		return Result{
			Code: code, PollSecret: secret, URL: domain.CLILoginURL(h.publicURL, code),
			ExpiresAt: expires, Interval: domain.CLIPollInterval,
		}, nil
	}
	return Result{}, domain.ErrTooManyCLILogins
}
