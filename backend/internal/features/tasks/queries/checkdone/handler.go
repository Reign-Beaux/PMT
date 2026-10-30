// Package checkdone consulta si todas las tareas de una etapa están terminadas.
package checkdone

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, stageID uuid.UUID) ([]domain.Task, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle indica si todas las tareas de la etapa están en done. Una etapa sin
// tareas cumple la condición.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID) (bool, error) {
	tasks, err := h.reader.List(ctx, stageID)
	if err != nil {
		return false, err
	}

	for _, t := range tasks {
		if t.Status != domain.StatusDone {
			return false, nil
		}
	}
	return true, nil
}
