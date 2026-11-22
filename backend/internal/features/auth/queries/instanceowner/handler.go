// Package instanceowner consulta al dueño de la instancia: los tokens de agente
// actúan con su membresía y adopta los proyectos sin owner.
package instanceowner

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
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

// Handle devuelve el ID del dueño de la instancia, o uuid.Nil si todavía no
// existe.
func (h *Handler) Handle(ctx context.Context) (uuid.UUID, error) {
	owner, err := h.reader.InstanceOwner(ctx)
	if errors.Is(err, domain.ErrUserNotFound) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	return owner.ID, nil
}
