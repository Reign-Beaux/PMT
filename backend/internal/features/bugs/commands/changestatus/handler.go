// Package changestatus cambia el status de un bug.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Bug, error)
	ChangeStatus(ctx context.Context, b domain.Bug, change domain.StatusChange) (domain.Bug, error)
}

// ProjectChecker confirma que un proyecto está en working. Lo implementa la
// feature projects, conectada en app.
type ProjectChecker interface {
	EnsureWorking(ctx context.Context, projectID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
	Reason string        `json:"reason"`
	Actor  kernel.Actor  `json:"-"`
}

// Result es el bug con su nuevo status.
type Result struct {
	ID               uuid.UUID     `json:"id"`
	ProjectID        uuid.UUID     `json:"project_id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	StepsToReproduce string        `json:"steps_to_reproduce"`
	ProductVersion   string        `json:"product_version"`
	Status           domain.Status `json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo     Repository
	projects ProjectChecker
}

// New crea el handler.
func New(repo Repository, projects ProjectChecker) *Handler {
	return &Handler{repo: repo, projects: projects}
}

// Handle cambia el status del bug. Exige el proyecto en working (regla 15).
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	bug, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	from := bug.Status
	if err := bug.ChangeStatus(cmd.Status, cmd.Actor, cmd.Reason); err != nil {
		return Result{}, err
	}
	if err := h.projects.EnsureWorking(ctx, bug.ProjectID); err != nil {
		return Result{}, err
	}

	statusChange := domain.NewStatusChange(bug.ID, from, bug.Status, cmd.Actor, cmd.Reason)
	updated, err := h.repo.ChangeStatus(ctx, bug, statusChange)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
