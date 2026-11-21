// Package getinvitation dice a quién es una invitación y si todavía vale. Es
// pública: el token del enlace es la credencial.
package getinvitation

import (
	"context"
	"time"

	"pmt/internal/features/auth/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	InvitationStatus(ctx context.Context, tokenHash []byte, now time.Time) (domain.InvitationStatus, error)
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

// Handle devuelve el correo y si se puede aceptar: sin usar, sin vencer y sin un
// usuario con ese correo.
func (h *Handler) Handle(ctx context.Context, token string) (domain.InvitationStatus, error) {
	return h.reader.InvitationStatus(ctx, domain.HashToken(token), h.now())
}
