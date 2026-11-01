// Package checkdone hace el recuento de las tareas de una etapa para cerrarla.
package checkdone

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

// Handle cuenta las tareas terminadas y las pendientes de la etapa. Una cancelada
// no cuenta, igual que una retirada.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID) (domain.Closure, error) {
	tasks, err := h.reader.List(ctx, stageID)
	if err != nil {
		return domain.Closure{}, err
	}

	var closure domain.Closure
	for _, t := range tasks {
		switch t.Status {
		case domain.StatusDone:
			closure.Done++
		case domain.StatusCancelled:
		default:
			closure.Pending++
		}
	}
	return closure, nil
}
