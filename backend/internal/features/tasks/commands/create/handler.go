// Package create registra una tarea nueva en una etapa.
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, t domain.Task) (domain.Task, error)
}

// StageChecker confirma que las tareas de una etapa se pueden editar
// directamente. Lo implementa la feature stages, conectada en app.
type StageChecker interface {
	EnsureTasksEditable(ctx context.Context, stageID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	StageID     uuid.UUID `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

// Result es la tarea creada.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Assignee    string        `json:"assignee,omitempty"`
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

// Handle crea la tarea.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := h.stages.EnsureTasksEditable(ctx, cmd.StageID); err != nil {
		return Result{}, err
	}
	t, err := domain.NewTask(cmd.StageID, cmd.Title, cmd.Description)
	if err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, t)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}
