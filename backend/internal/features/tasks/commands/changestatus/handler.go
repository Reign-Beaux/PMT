// Package changestatus cambia el status de una tarea.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	ChangeStatus(ctx context.Context, task domain.Task, change domain.StatusChange) (domain.Task, error)
}

// StageChecker da el equipo de una etapa y dice qué admiten sus tareas según el
// status de la etapa y del proyecto. Lo implementa la feature stages.
type StageChecker interface {
	Team(ctx context.Context, id uuid.UUID) (kernel.Team, error)
	EnsureWorkOpen(ctx context.Context, id uuid.UUID) error
	EnsureTaskCancellable(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
	Reason string        `json:"reason"`
	Actor  kernel.Actor  `json:"-"`
}

// Result es la tarea con su nuevo status y su última versión.
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

// Handle cambia el status de la tarea si su etapa y su proyecto lo permiten. Una
// tarea retirada no se encuentra.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	task, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	from := task.Status
	if err := task.EnsureProfile(cmd.Status, cmd.Actor); err != nil {
		return Result{}, err
	}
	team, err := h.stages.Team(ctx, task.StageID)
	if err != nil {
		return Result{}, err
	}
	if err := task.ChangeStatus(cmd.Status, cmd.Actor, cmd.Reason, team); err != nil {
		return Result{}, err
	}
	if err := h.ensureRules(ctx, from, task); err != nil {
		return Result{}, err
	}

	statusChange := domain.NewStatusChange(task.ID, from, task.Status, cmd.Actor, cmd.Reason)
	updated, err := h.repo.ChangeStatus(ctx, task, statusChange)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}

// ensureRules aplica las reglas entre status, que dependen de la etapa y el proyecto.
func (h *Handler) ensureRules(ctx context.Context, from domain.Status, task domain.Task) error {
	if domain.IsCancelOrReopen(from, task.Status) {
		return h.stages.EnsureTaskCancellable(ctx, task.StageID)
	}
	return h.stages.EnsureWorkOpen(ctx, task.StageID)
}
