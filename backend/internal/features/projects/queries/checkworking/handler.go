// Package checkworking consulta si un proyecto está en working.
package checkworking

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle indica si el proyecto está en working. Si no existe, devuelve un error de negocio.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) (bool, error) {
	p, err := h.reader.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return p.IsWorking(), nil
}
