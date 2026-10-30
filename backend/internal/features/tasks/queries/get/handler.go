// Package get consulta una tarea.
package get

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
}

// Result es la tarea consultada.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve la tarea.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) (Result, error) {
	t, err := h.reader.Get(ctx, id)
	if err != nil {
		return Result{}, err
	}
	return Result(t), nil
}
