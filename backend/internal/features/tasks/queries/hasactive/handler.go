// Package hasactive consulta si una etapa tiene tareas sin retirar.
package hasactive

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

// Handle indica si la etapa tiene tareas sin retirar.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID) (bool, error) {
	tasks, err := h.reader.List(ctx, stageID)
	if err != nil {
		return false, err
	}
	return len(tasks) > 0, nil
}
