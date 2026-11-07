// Package revise crea una versión nueva de un plan de pruebas.
package revise

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Plan, error)
	Revise(ctx context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error)
}

// TaskChecker da la etapa de una tarea. Lo implementa la feature tasks.
type TaskChecker interface {
	StageOf(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID         uuid.UUID   `json:"-"`
	HappyPath  string      `json:"happy_path"`
	ErrorCases string      `json:"error_cases"`
	Reason     string      `json:"reason"`
	TaskIDs    []uuid.UUID `json:"task_ids"`

	Actor kernel.Actor `json:"-"`
}

// Result es el plan con su versión nueva.
type Result struct {
	ID      uuid.UUID     `json:"id"`
	Status  domain.Status `json:"status"`
	Version int           `json:"version"`
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

// Handle valida con una copia, antes de consultar otra feature, y decide de nuevo
// con lo leído tras bloquear.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureQAP(cmd.Actor); err != nil {
		return Result{}, err
	}
	plan, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	stageID := plan.StageID
	apply := func(p *domain.Plan) error {
		return p.Revise(cmd.HappyPath, cmd.ErrorCases, cmd.Reason, cmd.TaskIDs, cmd.Actor)
	}
	if err := apply(&plan); err != nil {
		return Result{}, err
	}
	for _, id := range cmd.TaskIDs {
		taskStage, err := h.tasks.StageOf(ctx, id)
		if err == nil && taskStage != stageID {
			err = domain.ErrTaskOutsideStage
		}
		if err != nil {
			return Result{}, kernel.Prefix("tarea "+id.String()+": ", err)
		}
	}

	revised, err := h.repo.Revise(ctx, cmd.ID, apply)
	if err != nil {
		return Result{}, err
	}
	return Result{ID: revised.ID, Status: revised.Status, Version: revised.Latest.Version}, nil
}
