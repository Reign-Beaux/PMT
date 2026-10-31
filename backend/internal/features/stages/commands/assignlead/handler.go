// Package assignlead fija o reemplaza el TL responsable de una etapa.
package assignlead

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	UpdateLead(ctx context.Context, id uuid.UUID, lead string) error
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

// Result es el equipo de la etapa.
type Result struct {
	ID      uuid.UUID `json:"id"`
	Lead    string    `json:"lead"`
	Members []string  `json:"members"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	agents AgentChecker
}

// New crea el handler.
func New(repo Repository, agents AgentChecker) *Handler {
	return &Handler{repo: repo, agents: agents}
}

// Handle asigna el TL. No es un cambio de contenido: aplica en cualquier status.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanAssignLead(cmd.Actor); err != nil {
		return Result{}, err
	}
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	profile, err := h.agents.Profile(ctx, cmd.Agent)
	if err != nil {
		return Result{}, err
	}
	if profile != kernel.ProfileTL {
		return Result{}, domain.ErrAgentNotLead
	}
	if err := h.repo.UpdateLead(ctx, s.ID, cmd.Agent); err != nil {
		return Result{}, err
	}
	return Result{ID: s.ID, Lead: cmd.Agent, Members: append([]string{}, s.Members...)}, nil
}
