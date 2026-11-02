// Package create registra un bug nuevo en un proyecto.
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, b domain.Bug) (domain.Bug, error)
}

// ProjectChecker confirma que un proyecto está en working. Lo implementa la
// feature projects, conectada en app.
type ProjectChecker interface {
	EnsureWorking(ctx context.Context, projectID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID        uuid.UUID    `json:"-"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	StepsToReproduce string       `json:"steps_to_reproduce"`
	ProductVersion   string       `json:"product_version"`
	Actor            kernel.Actor `json:"-"`
}

// Result es el bug creado.
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

// Handle crea el bug. El perfil se comprueba antes de consultar otra feature.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanCreate(cmd.Actor); err != nil {
		return Result{}, err
	}
	if err := h.projects.EnsureWorking(ctx, cmd.ProjectID); err != nil {
		return Result{}, err
	}
	b, err := domain.NewBug(cmd.ProjectID, cmd.Title, cmd.Description, cmd.StepsToReproduce, cmd.ProductVersion)
	if err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, b)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}
