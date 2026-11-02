// Package releaseproject libera un proyecto en planning junto con sus etapas.
package releaseproject

import (
	"context"

	"github.com/google/uuid"

	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	"pmt/internal/features/workflow/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Project(ctx context.Context, id uuid.UUID) (projects.Project, error)
	Stages(ctx context.Context, projectID uuid.UUID) ([]stages.Stage, error)
	Release(ctx context.Context, plan domain.ReleasePlan) error
}

// TaskChecker comprueba la regla 4. Lo implementa la feature tasks.
type TaskChecker interface {
	EnsureReleasable(ctx context.Context, stageID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID    `json:"-"`
	Reason string       `json:"reason"`
	Actor  kernel.Actor `json:"-"`
}

// Stage es una etapa liberada.
type Stage struct {
	ID     uuid.UUID     `json:"id"`
	Status stages.Status `json:"status"`
}

// Result es el proyecto liberado y las etapas que pasaron a stand_by.
type Result struct {
	ID     uuid.UUID       `json:"id"`
	Status projects.Status `json:"status"`
	Stages []Stage         `json:"stages"`
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

// Handle pasa el proyecto y sus etapas en planning a stand_by, todos o ninguno. Las
// etapas en otro status no se tocan, y las tareas siguen en backlog.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	p, err := h.repo.Project(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := p.EnsurePlanning(); err != nil {
		return Result{}, err
	}
	from := p.Status
	if err := p.ChangeStatus(projects.StatusStandBy, cmd.Actor, cmd.Reason); err != nil {
		return Result{}, err
	}

	current, err := h.repo.Stages(ctx, p.ID)
	if err != nil {
		return Result{}, err
	}
	if err := stages.EnsureHasActive(current); err != nil {
		return Result{}, err
	}

	plan := domain.ReleasePlan{Project: domain.ProjectMove{
		Project: p,
		Change:  projects.NewStatusChange(p.ID, from, p.Status, cmd.Actor, cmd.Reason),
	}}
	result := Result{ID: p.ID, Status: p.Status, Stages: []Stage{}}
	for _, s := range current {
		if s.Status != stages.StatusPlanning {
			continue
		}
		move, err := h.release(ctx, s, cmd)
		if err != nil {
			return Result{}, kernel.Prefix("etapa "+s.Name+": ", err)
		}
		plan.Stages = append(plan.Stages, move)
		result.Stages = append(result.Stages, Stage{ID: s.ID, Status: move.Stage.Status})
	}

	if err := h.repo.Release(ctx, plan); err != nil {
		return Result{}, err
	}
	return result, nil
}

// release prepara el paso de una etapa. La regla 4 también vale para el planning
// propio, que sale con su registro add.
func (h *Handler) release(ctx context.Context, s stages.Stage, cmd Command) (domain.StageMove, error) {
	from := s.Status
	if err := s.ChangeStatus(stages.StatusStandBy, cmd.Actor, cmd.Reason); err != nil {
		return domain.StageMove{}, err
	}
	if err := h.tasks.EnsureReleasable(ctx, s.ID); err != nil {
		return domain.StageMove{}, err
	}
	move := domain.StageMove{Stage: s, Change: stages.NewStatusChange(s.ID, from, s.Status, cmd.Actor, cmd.Reason)}
	if s.InOwnPlanning() {
		add, err := stages.NewChange(s, stages.ChangeAdd, s.AddReason, s.AddRequestedBy)
		if err != nil {
			return domain.StageMove{}, err
		}
		move.Add = &add
	}
	return move, nil
}
