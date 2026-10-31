// Package retire retira una etapa sin borrar nada.
package retire

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	Retire(ctx context.Context, change domain.Change) (domain.Change, error)
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

// Handle registra el retiro. Una etapa con tareas activas no se retira.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	stage, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
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

	saved, err := h.repo.Retire(ctx, change)
	if err != nil {
		return Result{}, err
	}
	return Result{RetiredID: saved.StageID, Version: saved.Version}, nil
}
