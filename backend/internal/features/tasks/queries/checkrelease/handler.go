// Package checkrelease consulta si una etapa tiene tareas activas para liberarse.
package checkrelease

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Reader es la lectura que necesita la operación. No devuelve lo retirado.
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

// Handle aplica la regla 4: la etapa necesita una tarea sin retirar y sin cancelar.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID) error {
	tasks, err := h.reader.List(ctx, stageID)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if t.Status != domain.StatusCancelled {
			return nil
		}
	}
	return domain.ErrStageNoActiveTasks
}
