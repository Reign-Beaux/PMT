// Package assign cambia el dueño de una tarea ya tomada.
package assign

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
	UpdateAssignee(ctx context.Context, task domain.Task) (domain.Task, error)
}

// StageChecker da el equipo de una etapa. Lo implementa la feature stages.
type StageChecker interface {
	Team(ctx context.Context, id uuid.UUID) (kernel.Team, error)
}

// AgentChecker da el perfil de un agente activo. Lo implementa la feature agents.
type AgentChecker interface {
	Profile(ctx context.Context, name string) (kernel.Profile, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID    uuid.UUID    `json:"-"`
	Agent string       `json:"agent"`
	Actor kernel.Actor `json:"-"`
}

// Result es la tarea con su nuevo dueño. DoneAt no se expone: solo las
// consultas de lectura traen done_at.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Assignee    string        `json:"assignee,omitempty"`
	DoneAt      *time.Time    `json:"-"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	stages StageChecker
	agents AgentChecker
}

// New crea el handler.
func New(repo Repository, stages StageChecker, agents AgentChecker) *Handler {
	return &Handler{repo: repo, stages: stages, agents: agents}
}

// Handle pone a la tarea un dueño del equipo. Solo lo hace el TL de la etapa.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if cmd.Actor.Profile != kernel.ProfileTL {
		return Result{}, domain.ErrAssignForbidden
	}
	task, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	team, err := h.stages.Team(ctx, task.StageID)
	if err != nil {
		return Result{}, err
	}
	if !team.IsLead(cmd.Actor) {
		return Result{}, kernel.ErrNotStageLead
	}
	if _, err := h.agents.Profile(ctx, cmd.Agent); err != nil {
		return Result{}, err
	}
	if !team.HasMember(cmd.Agent) {
		return Result{}, domain.ErrAssigneeNotMember
	}
	if err := task.Assign(cmd.Agent); err != nil {
		return Result{}, err
	}
	updated, err := h.repo.UpdateAssignee(ctx, task)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
