// Package retire retira una etapa sin borrar nada.
package retire

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	// RetireStage guarda el retiro y, si termina la versión 1, pasa el proyecto a
	// working en la misma transacción. Lo implementa la feature workflow.
	RetireStage(ctx context.Context, s domain.Stage, change domain.Change, actor kernel.Actor) (domain.Change, error)
}

// TaskChecker indica si una etapa tiene tareas sin retirar. Lo implementa la
// feature tasks, conectada en app.
type TaskChecker interface {
	HasActive(ctx context.Context, stageID uuid.UUID) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`

	Actor kernel.Actor `json:"-"`
}

// Result identifica el retiro.
type Result struct {
	RetiredID uuid.UUID `json:"retired_id"`
	Version   int       `json:"version"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo  Repository
	tasks TaskChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks}
}

// Handle registra el retiro. Una etapa con tareas sin retirar no se retira.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanPlan(cmd.Actor); err != nil {
		return Result{}, err
	}
	stage, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := stage.EnsureHistoryRule(); err != nil {
		return Result{}, err
	}
	hasActive, err := h.tasks.HasActive(ctx, stage.ID)
	if err != nil {
		return Result{}, err
	}
	if hasActive {
		return Result{}, domain.ErrStageHasActiveTasks
	}
	change, err := domain.NewChange(stage, domain.ChangeRetire, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	saved, err := h.repo.RetireStage(ctx, stage, change, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	return Result{RetiredID: saved.StageID, Version: saved.Version}, nil
}
