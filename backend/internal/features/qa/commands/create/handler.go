// Package create crea un plan de pruebas con su versión 1.
package create

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, p domain.Plan) (domain.Plan, error)
}

// TaskChecker da la etapa de una tarea. Lo implementa la feature tasks.
type TaskChecker interface {
	StageOf(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// StageChecker comprueba que la etapa exista. Lo implementa la feature stages.
type StageChecker interface {
	Check(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	StageID    uuid.UUID   `json:"-"`
	Title      string      `json:"title"`
	HappyPath  string      `json:"happy_path"`
	ErrorCases string      `json:"error_cases"`
	TaskIDs    []uuid.UUID `json:"task_ids"`

	Actor kernel.Actor `json:"-"`
}

// Result es el plan creado.
type Result struct {
	ID      uuid.UUID     `json:"id"`
	Status  domain.Status `json:"status"`
	Version int           `json:"version"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	tasks  TaskChecker
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker, stages StageChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks, stages: stages}
}

// Handle crea el plan. Las tareas deben ser de la etapa y estar sin retirar.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	plan, err := domain.NewPlan(cmd.StageID, cmd.Title, cmd.HappyPath, cmd.ErrorCases, cmd.TaskIDs, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	if err := h.stages.Check(ctx, cmd.StageID); err != nil {
		return Result{}, err
	}
	for _, id := range cmd.TaskIDs {
		stageID, err := h.tasks.StageOf(ctx, id)
		if err == nil && stageID != cmd.StageID {
			err = domain.ErrTaskOutsideStage
		}
		if err != nil {
			return Result{}, kernel.Prefix("tarea "+id.String()+": ", err)
		}
	}

	created, err := h.repo.Create(ctx, plan)
	if err != nil {
		return Result{}, err
	}
	return Result{ID: created.ID, Status: created.Status, Version: created.Latest.Version}, nil
}
