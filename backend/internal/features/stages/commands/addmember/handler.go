// Package addmember añade un dev al equipo de una etapa.
package addmember

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	AddMember(ctx context.Context, id uuid.UUID, agent, addedBy string) error
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

// Handle añade al dev. Solo lo hace el TL de la etapa.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureLeadProfile(cmd.Actor); err != nil {
		return Result{}, err
	}
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := s.EnsureLead(cmd.Actor); err != nil {
		return Result{}, err
	}
	profile, err := h.agents.Profile(ctx, cmd.Agent)
	if err != nil {
		return Result{}, err
	}
	if !profile.IsDev() {
		return Result{}, domain.ErrAgentNotDev
	}
	if err := h.repo.AddMember(ctx, s.ID, cmd.Agent, cmd.Actor.Name); err != nil {
		return Result{}, err
	}
	return Result{ID: s.ID, Lead: s.Lead, Members: slices.Concat(s.Members, []string{cmd.Agent})}, nil
}
