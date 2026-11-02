// Package checkstatus consulta qué admiten las etapas y tareas de un proyecto
// según su status.
package checkstatus

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

// AcceptsWork indica si sus etapas y tareas pueden avanzar. Si el proyecto no
// existe, devuelve un error de negocio.
func (h *Handler) AcceptsWork(ctx context.Context, id uuid.UUID) (bool, error) {
	p, err := h.reader.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return p.AcceptsWork(), nil
}

// IsFrozen indica si congela sus etapas y tareas. Si el proyecto no existe,
// devuelve un error de negocio.
func (h *Handler) IsFrozen(ctx context.Context, id uuid.UUID) (bool, error) {
	p, err := h.reader.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return p.IsFrozen(), nil
}
