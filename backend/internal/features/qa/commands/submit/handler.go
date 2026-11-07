// Package submit envía un plan de pruebas al TL.
package submit

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Decide(ctx context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID    uuid.UUID
	Actor kernel.Actor
}

// Result es el plan enviado.
type Result struct {
	ID      uuid.UUID     `json:"id"`
	Status  domain.Status `json:"status"`
	Version int           `json:"version"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle envía el plan. Decide con lo leído tras bloquear.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureQAP(cmd.Actor); err != nil {
		return Result{}, err
	}
	plan, err := h.repo.Decide(ctx, cmd.ID, func(p *domain.Plan) error {
		return p.Submit(cmd.Actor)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{ID: plan.ID, Status: plan.Status, Version: plan.Latest.Version}, nil
}
