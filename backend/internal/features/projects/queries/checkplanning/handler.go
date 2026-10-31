// Package checkplanning consulta si un proyecto está en planning.
package checkplanning

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

// Handle devuelve un error de negocio si el proyecto no existe o no está en planning.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) error {
	p, err := h.reader.Get(ctx, id)
	if err != nil {
		return err
	}
	return p.EnsurePlanning()
}
