// Package delete elimina una etapa.
package delete

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	HasChanges(ctx context.Context, id uuid.UUID) (bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProjectChecker confirma el status de un proyecto. Lo implementa la feature
// projects, conectada en app.
type ProjectChecker interface {
	EnsurePlanning(ctx context.Context, projectID uuid.UUID) error
	EnsureWorking(ctx context.Context, projectID uuid.UUID) error
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

// Handle elimina la etapa.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID, actor kernel.Actor) error {
	if err := domain.EnsureCanPlan(actor); err != nil {
		return err
	}
	s, err := h.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.InOwnPlanning() {
		if err := h.projects.EnsureWorking(ctx, s.ProjectID); err != nil {
			return err
		}
		return h.repo.Delete(ctx, id)
	}
	if err := h.projects.EnsurePlanning(ctx, s.ProjectID); err != nil {
		return err
	}
	hasChanges, err := h.repo.HasChanges(ctx, id)
	if err != nil {
		return err
	}
	if hasChanges {
		return domain.ErrStageHasChanges
	}
	return h.repo.Delete(ctx, id)
}
