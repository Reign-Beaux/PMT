// Package getclilogin muestra a la persona el código que su CLI pidió, para que
// compruebe que coincide antes de autorizarlo.
package getclilogin

import (
	"context"
	"time"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	domain.UserFinder
	CLILoginByCode(ctx context.Context, code string) (domain.CLILogin, error)
}

// Query contiene los datos de entrada.
type Query struct {
	Code  string
	Actor kernel.Actor
}

// Item es el código como lo ve la pantalla /cli.
type Item struct {
	Code      string    `json:"code"`
	Machine   string    `json:"machine"`
	TokenName string    `json:"token_name"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
	now    func() time.Time
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader, now: time.Now}
}

// Handle devuelve el código, la máquina y su estado. Es una operación de cuenta:
// solo con el perfil User.
func (h *Handler) Handle(ctx context.Context, q Query) (Item, error) {
	if _, err := domain.CurrentUser(ctx, h.reader, q.Actor); err != nil {
		return Item{}, err
	}
	code, err := domain.NormalizeCLICode(q.Code)
	if err != nil {
		return Item{}, err
	}
	login, err := h.reader.CLILoginByCode(ctx, code)
	if err != nil {
		return Item{}, err
	}
	return ItemOf(login, h.now()), nil
}

// ItemOf arma el Item de un inicio de sesión a una hora dada.
func ItemOf(login domain.CLILogin, now time.Time) Item {
	return Item{
		Code: login.Code, Machine: login.Machine, TokenName: domain.CLITokenName(login.Machine),
		Status: login.Status(now), ExpiresAt: login.ExpiresAt,
	}
}
