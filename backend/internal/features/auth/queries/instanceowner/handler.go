// Package instanceowner consulta al dueño de la instancia: los tokens de agente
// actúan con su membresía y adopta los proyectos sin owner.
package instanceowner

import (
	"context"
	"errors"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	InstanceOwner(ctx context.Context) (domain.User, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve al dueño de la instancia, o un UserRef vacío si todavía no
// existe.
func (h *Handler) Handle(ctx context.Context) (kernel.UserRef, error) {
	owner, err := h.reader.InstanceOwner(ctx)
	if errors.Is(err, domain.ErrUserNotFound) {
		return kernel.UserRef{}, nil
	}
	if err != nil {
		return kernel.UserRef{}, err
	}
	return owner.Ref(), nil
}
