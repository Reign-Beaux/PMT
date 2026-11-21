// Package removelead deja una etapa sin TL responsable.
package removelead

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	ClearLead(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID    uuid.UUID
	Actor kernel.Actor
}

// Result es el equipo de la etapa.
type Result struct {
	ID      uuid.UUID `json:"id"`
	Lead    string    `json:"lead"`
	Members []string  `json:"members"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle quita el TL. Solo PP o el usuario, y solo con la etapa sin iniciar. El
// equipo no cambia.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanRemoveLead(cmd.Actor); err != nil {
		return Result{}, err
	}
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := s.EnsureLeadRemovable(); err != nil {
		return Result{}, err
	}
	if err := h.repo.ClearLead(ctx, s.ID); err != nil {
		return Result{}, err
	}
	return Result{ID: s.ID, Lead: "", Members: append([]string{}, s.Members...)}, nil
}
