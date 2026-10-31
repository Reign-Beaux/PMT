// Package update cambia los datos de una tarea.
package update

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	HasChanges(ctx context.Context, id uuid.UUID) (bool, error)
	Update(ctx context.Context, t domain.Task) (domain.Task, error)
}

// StageChecker confirma que las tareas de una etapa se pueden editar
// directamente. Lo implementa la feature stages, conectada en app.
type StageChecker interface {
	EnsureTasksEditable(ctx context.Context, stageID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

// Result es la tarea actualizada.
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
	repo   Repository
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, stages StageChecker) *Handler {
	return &Handler{repo: repo, stages: stages}
}

// Handle actualiza la tarea.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	t, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := h.stages.EnsureTasksEditable(ctx, t.StageID); err != nil {
		return Result{}, err
	}
	hasChanges, err := h.repo.HasChanges(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if hasChanges {
		return Result{}, domain.ErrTaskHasChanges
	}
	if err := t.Update(cmd.Title, cmd.Description); err != nil {
		return Result{}, err
	}

	updated, err := h.repo.Update(ctx, t)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
