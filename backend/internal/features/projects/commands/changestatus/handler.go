// Package changestatus cambia el status de un proyecto.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
	ChangeStatus(ctx context.Context, p domain.Project, change domain.StatusChange) (domain.Project, error)
	HasReachedWorking(ctx context.Context, id uuid.UUID) (bool, error)
}

// StageProgress comprueba que las etapas de un proyecto permitan el cambio y
// responde con el error de negocio. Lo implementa la feature stages.
type StageProgress interface {
	EnsureReleasable(ctx context.Context, projectID uuid.UUID) error
	EnsureStartable(ctx context.Context, projectID uuid.UUID) error
	EnsureClosed(ctx context.Context, projectID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
	Reason string        `json:"reason"`
	Actor  kernel.Actor  `json:"-"`
}

// Result es el proyecto con su nuevo status.
type Result struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Status         domain.Status `json:"status"`
	ProductVersion string        `json:"product_version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	stages StageProgress
}

// New crea el handler.
func New(repo Repository, stages StageProgress) *Handler {
	return &Handler{repo: repo, stages: stages}
}

// Handle cambia el status del proyecto si sus etapas lo permiten.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanChangeStatus(cmd.Actor); err != nil {
		return Result{}, err
	}
	p, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	from := p.Status
	if err := p.ChangeStatus(cmd.Status, cmd.Actor, cmd.Reason); err != nil {
		return Result{}, err
	}
	if err := h.ensureStages(ctx, from, p); err != nil {
		return Result{}, err
	}

	statusChange := domain.NewStatusChange(p.ID, from, p.Status, cmd.Actor, cmd.Reason)
	updated, err := h.repo.ChangeStatus(ctx, p, statusChange)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}

// ensureStages aplica las reglas 11 a 14. Las de liberar e iniciar solo valen
// desde el origen que nombran. stopped → working, la única llegada manual a working,
// solo se comprueba si el proyecto nunca estuvo ahí: si no, es reanudar.
func (h *Handler) ensureStages(ctx context.Context, from domain.Status, p domain.Project) error {
	switch {
	case from == domain.StatusPlanning && p.Status == domain.StatusStandBy:
		return h.stages.EnsureReleasable(ctx, p.ID)
	case from == domain.StatusStandBy && p.Status == domain.StatusInDev:
		return h.stages.EnsureStartable(ctx, p.ID)
	case from == domain.StatusStopped && p.Status == domain.StatusWorking:
		reached, err := h.repo.HasReachedWorking(ctx, p.ID)
		if err != nil || reached {
			return err
		}
		return h.stages.EnsureClosed(ctx, p.ID)
	}
	return nil
}
