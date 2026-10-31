// Package create registra una etapa nueva al final de su proyecto.
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, s domain.Stage) (domain.Stage, error)
}

// ProjectChecker confirma que un proyecto está en planning. Lo implementa la
// feature projects, conectada en app.
type ProjectChecker interface {
	EnsurePlanning(ctx context.Context, projectID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID   uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// Result es la etapa creada.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
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

// Handle crea la etapa.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := h.projects.EnsurePlanning(ctx, cmd.ProjectID); err != nil {
		return Result{}, err
	}
	s, err := domain.NewStage(cmd.ProjectID, cmd.Name, cmd.Description)
	if err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, s)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}
