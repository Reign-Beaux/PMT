// Package revise crea una versión nueva de un plan de desarrollo.
package revise

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/devplans/domain"
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

// Checker comprueba que un elemento exista. Lo implementan stages y bugs.
type Checker interface {
	Check(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID                 uuid.UUID   `json:"-"`
	TaskIDs            []uuid.UUID `json:"task_ids"`
	Scope              string      `json:"scope"`
	TechnicalDecisions string      `json:"technical_decisions"`
	ScreenStates       string      `json:"screen_states"`
	EdgeCases          string      `json:"edge_cases"`
	ErrorHandling      string      `json:"error_handling"`
	AcceptanceCriteria string      `json:"acceptance_criteria"`
	Reason             string      `json:"reason"`

	Actor kernel.Actor `json:"-"`
}

// Result es el plan con su versión nueva.
type Result struct {
	ID      uuid.UUID `json:"id"`
	Version int       `json:"version"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	tasks  TaskChecker
	stages Checker
	bugs   Checker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker, stages, bugs Checker) *Handler {
	return &Handler{repo: repo, tasks: tasks, stages: stages, bugs: bugs}
}

// Handle valida con una copia, antes de consultar otra feature, y decide de nuevo
// con lo leído tras bloquear. El perfil se comprueba antes de leer nada.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureDP(cmd.Actor); err != nil {
		return Result{}, err
	}
	plan, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	content := domain.Content{
		Scope:              cmd.Scope,
		TechnicalDecisions: cmd.TechnicalDecisions,
		ScreenStates:       cmd.ScreenStates,
		EdgeCases:          cmd.EdgeCases,
		ErrorHandling:      cmd.ErrorHandling,
		AcceptanceCriteria: cmd.AcceptanceCriteria,
	}
	apply := func(p *domain.Plan) error {
		return p.Revise(content, cmd.Reason, cmd.TaskIDs, cmd.Actor)
	}
	if err := apply(&plan); err != nil {
		return Result{}, err
	}
	if err := h.ensureReferences(ctx, plan); err != nil {
		return Result{}, err
	}

	revised, err := h.repo.Revise(ctx, cmd.ID, apply)
	if err != nil {
		return Result{}, err
	}
	return Result{ID: revised.ID, Version: revised.Latest.Version}, nil
}

func (h *Handler) ensureReferences(ctx context.Context, plan domain.Plan) error {
	if plan.BugID != nil {
		return h.bugs.Check(ctx, *plan.BugID)
	}
	if err := h.stages.Check(ctx, *plan.StageID); err != nil {
		return err
	}
	for _, id := range plan.TaskIDs {
		stageID, err := h.tasks.StageOf(ctx, id)
		if err == nil && stageID != *plan.StageID {
			err = domain.ErrTaskOutsideStage
		}
		if err != nil {
			return kernel.Prefix("tarea "+id.String()+": ", err)
		}
	}
	return nil
}
