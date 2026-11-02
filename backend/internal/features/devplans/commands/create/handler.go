// Package create crea un plan de desarrollo con su versión 1.
package create

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/devplans/domain"
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

// Checker comprueba que un elemento exista. Lo implementan stages y bugs.
type Checker interface {
	Check(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada: StageID con TaskIDs, o BugID.
type Command struct {
	StageID            *uuid.UUID  `json:"stage_id"`
	BugID              *uuid.UUID  `json:"bug_id"`
	TaskIDs            []uuid.UUID `json:"task_ids"`
	Scope              string      `json:"scope"`
	TechnicalDecisions string      `json:"technical_decisions"`
	ScreenStates       string      `json:"screen_states"`
	EdgeCases          string      `json:"edge_cases"`
	ErrorHandling      string      `json:"error_handling"`
	AcceptanceCriteria string      `json:"acceptance_criteria"`

	Actor kernel.Actor `json:"-"`
}

// Result es el plan creado.
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

// Handle crea el plan. La etapa o el bug deben existir, y las tareas ser de la etapa
// y estar sin retirar.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	content := domain.Content{
		Scope:              cmd.Scope,
		TechnicalDecisions: cmd.TechnicalDecisions,
		ScreenStates:       cmd.ScreenStates,
		EdgeCases:          cmd.EdgeCases,
		ErrorHandling:      cmd.ErrorHandling,
		AcceptanceCriteria: cmd.AcceptanceCriteria,
	}
	plan, err := domain.NewPlan(cmd.StageID, cmd.BugID, cmd.TaskIDs, content, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	if err := h.ensureReferences(ctx, cmd); err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, plan)
	if err != nil {
		return Result{}, err
	}
	return Result{ID: created.ID, Version: created.Latest.Version}, nil
}

func (h *Handler) ensureReferences(ctx context.Context, cmd Command) error {
	if cmd.BugID != nil {
		return h.bugs.Check(ctx, *cmd.BugID)
	}
	if err := h.stages.Check(ctx, *cmd.StageID); err != nil {
		return err
	}
	for _, id := range cmd.TaskIDs {
		stageID, err := h.tasks.StageOf(ctx, id)
		if err == nil && stageID != *cmd.StageID {
			err = domain.ErrTaskOutsideStage
		}
		if err != nil {
			return kernel.Prefix("tarea "+id.String()+": ", err)
		}
	}
	return nil
}
