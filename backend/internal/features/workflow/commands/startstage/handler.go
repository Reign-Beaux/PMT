// Package startstage inicia una etapa: la toma su TL y libera sus tareas.
package startstage

import (
	"context"

	"github.com/google/uuid"

	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/features/workflow/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Project(ctx context.Context, id uuid.UUID) (projects.Project, error)
	Stage(ctx context.Context, id uuid.UUID) (stages.Stage, error)
	Tasks(ctx context.Context, stageID uuid.UUID) ([]tasks.Task, error)
	Start(ctx context.Context, plan domain.StartPlan) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID    `json:"-"`
	Reason string       `json:"reason"`
	Actor  kernel.Actor `json:"-"`
}

// Task es una tarea liberada.
type Task struct {
	ID     uuid.UUID    `json:"id"`
	Status tasks.Status `json:"status"`
}

// Result es la etapa iniciada, el status de su proyecto y las tareas liberadas.
type Result struct {
	ID            uuid.UUID       `json:"id"`
	Status        stages.Status   `json:"status"`
	ProjectID     uuid.UUID       `json:"project_id"`
	ProjectStatus projects.Status `json:"project_status"`
	Tasks         []Task          `json:"tasks"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle pasa la etapa a in_dev, sus tareas en backlog a ready y el proyecto a
// in_dev si sigue en stand_by, todo o nada. Las demás tareas no se tocan.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	s, err := h.repo.Stage(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if s.Status != stages.StatusStandBy {
		return Result{}, stages.ErrStageNotStandBy
	}
	from := s.Status
	if err := s.ChangeStatus(stages.StatusInDev, cmd.Actor, cmd.Reason); err != nil {
		return Result{}, err
	}

	p, err := h.repo.Project(ctx, s.ProjectID)
	if err != nil {
		return Result{}, err
	}
	plan := domain.StartPlan{Stage: domain.StageMove{
		Stage:  s,
		Change: stages.NewStatusChange(s.ID, from, s.Status, cmd.Actor, cmd.Reason),
	}}
	if plan.Project, err = moveProject(&p, s, cmd.Actor); err != nil {
		return Result{}, err
	}

	current, err := h.repo.Tasks(ctx, s.ID)
	if err != nil {
		return Result{}, err
	}
	result := Result{ID: s.ID, Status: s.Status, ProjectID: p.ID, ProjectStatus: p.Status, Tasks: []Task{}}
	for _, t := range current {
		if t.Status != tasks.StatusBacklog {
			continue
		}
		taskFrom := t.Status
		if err := t.ChangeStatus(tasks.StatusReady, cmd.Actor, cmd.Reason, s.Team()); err != nil {
			return Result{}, kernel.Prefix("tarea "+t.ID.String()+": ", err)
		}
		plan.Tasks = append(plan.Tasks, domain.TaskMove{Task: t, Change: tasks.NewStatusChange(t.ID, taskFrom, t.Status, cmd.Actor, cmd.Reason)})
		result.Tasks = append(result.Tasks, Task{ID: t.ID, Status: t.Status})
	}

	if err := h.repo.Start(ctx, plan); err != nil {
		return Result{}, err
	}
	return result, nil
}

// moveProject exige un proyecto que admita iniciar etapas y prepara su paso
// automático si sigue en stand_by, con el TL como actor.
func moveProject(p *projects.Project, s stages.Stage, actor kernel.Actor) (*domain.ProjectMove, error) {
	switch p.Status {
	case projects.StatusStandBy:
		from := p.Status
		p.StartWork()
		return &domain.ProjectMove{
			Project: *p,
			Change:  projects.NewStatusChange(p.ID, from, p.Status, actor, "inicio de la etapa "+s.Name),
		}, nil
	case projects.StatusInDev, projects.StatusWorking:
		return nil, nil
	}
	return nil, stages.ErrStageProjectNotReady
}
