// Package update cambia los datos de una etapa.
package update

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	HasChanges(ctx context.Context, id uuid.UUID) (bool, error)
	Update(ctx context.Context, s domain.Stage) (domain.Stage, error)
}

// ProjectChecker confirma el status de un proyecto. Lo implementa la feature
// projects, conectada en app.
type ProjectChecker interface {
	EnsurePlanning(ctx context.Context, projectID uuid.UUID) error
	EnsureWorking(ctx context.Context, projectID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// Result es la etapa actualizada.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	AddReason      string `json:"add_reason,omitempty"`
	AddRequestedBy string `json:"add_requested_by,omitempty"`

	Lead    string   `json:"lead,omitempty"`
	Members []string `json:"members,omitempty"`
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

// Handle actualiza la etapa.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := h.ensureEditable(ctx, s); err != nil {
		return Result{}, err
	}
	if err := s.Update(cmd.Name, cmd.Description); err != nil {
		return Result{}, err
	}

	updated, err := h.repo.Update(ctx, s)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}

// ensureEditable aplica la regla de planning propio o la del proyecto, según la etapa.
func (h *Handler) ensureEditable(ctx context.Context, s domain.Stage) error {
	if s.InOwnPlanning() {
		return h.projects.EnsureWorking(ctx, s.ProjectID)
	}
	if err := h.projects.EnsurePlanning(ctx, s.ProjectID); err != nil {
		return err
	}
	hasChanges, err := h.repo.HasChanges(ctx, s.ID)
	if err != nil {
		return err
	}
	if hasChanges {
		return domain.ErrStageHasChanges
	}
	return nil
}
