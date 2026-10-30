// Package list consulta las tareas de una etapa.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, stageID uuid.UUID) ([]domain.Task, error)
}

// StageChecker confirma que una etapa existe. Lo implementa la feature stages,
// que responde con un error de negocio si no la encuentra.
type StageChecker interface {
	Check(ctx context.Context, id uuid.UUID) error
}

// Item es una tarea de la lista.
type Item struct {
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
	stages StageChecker
}

// New crea el handler.
func New(reader Reader, stages StageChecker) *Handler {
	return &Handler{reader: reader, stages: stages}
}

// Handle devuelve las tareas de la etapa, de la más antigua a la más reciente.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID) ([]Item, error) {
	if err := h.stages.Check(ctx, stageID); err != nil {
		return nil, err
	}

	tasks, err := h.reader.List(ctx, stageID)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, Item(t))
	}
	return items, nil
}
