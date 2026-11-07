// Package approve aprueba la última versión de un plan de pruebas.
package approve

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Plan, error)
	Decide(ctx context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error)
}

// StageChecker da el equipo de una etapa. Lo implementa la feature stages.
type StageChecker interface {
	Team(ctx context.Context, stageID uuid.UUID) (kernel.Team, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID    uuid.UUID
	Actor kernel.Actor
}

// Result es el plan aprobado.
type Result struct {
	ID         uuid.UUID     `json:"id"`
	Status     domain.Status `json:"status"`
	Version    int           `json:"version"`
	ReviewedBy string        `json:"reviewed_by"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, stages StageChecker) *Handler {
	return &Handler{repo: repo, stages: stages}
}

// Handle valida status, perfil y ámbito, en ese orden. El ámbito solo se consulta
// si el perfil ya pasó.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureTL(cmd.Actor); err != nil {
		return Result{}, err
	}
	plan, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := plan.EnsureDecidable(cmd.Actor); err != nil {
		return Result{}, err
	}
	team, err := h.stages.Team(ctx, plan.StageID)
	if err != nil {
		return Result{}, err
	}
	if !team.IsLead(cmd.Actor) {
		return Result{}, kernel.ErrNotStageLead
	}

	approved, err := h.repo.Decide(ctx, cmd.ID, func(p *domain.Plan) error {
		return p.Approve(cmd.Actor)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{ID: approved.ID, Status: approved.Status, Version: approved.Latest.Version, ReviewedBy: approved.ReviewedBy}, nil
}
