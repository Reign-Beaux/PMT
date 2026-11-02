// Package delete elimina una tarea.
package delete

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	HasChanges(ctx context.Context, id uuid.UUID) (bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// StageChecker confirma que las tareas de una etapa se pueden editar
// directamente. Lo implementa la feature stages, conectada en app.
type StageChecker interface {
	EnsureTasksEditable(ctx context.Context, stageID uuid.UUID) error
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, stages StageChecker) *Handler {
	return &Handler{repo: repo, stages: stages}
}

// Handle elimina la tarea.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) error {
	t, err := h.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := h.stages.EnsureTasksEditable(ctx, t.StageID); err != nil {
		return err
	}
	hasChanges, err := h.repo.HasChanges(ctx, id)
	if err != nil {
		return err
	}
	if hasChanges {
		return domain.ErrTaskHasChanges
	}
	return h.repo.Delete(ctx, id)
}
