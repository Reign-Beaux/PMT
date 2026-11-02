// Package get confirma que un bug exista.
package get

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Bug, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve un error de negocio si el bug no existe.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) error {
	_, err := h.reader.Get(ctx, id)
	return err
}
