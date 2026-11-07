// Package removemember quita un dev del equipo de una etapa.
package removemember

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
	RemoveMember(ctx context.Context, id uuid.UUID, agent string) error
}

// TaskChecker indica si un agente tiene tareas abiertas en una etapa. Lo implementa
// la feature tasks.
type TaskChecker interface {
	HasOpenAssigned(ctx context.Context, stageID uuid.UUID, agent string) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID    uuid.UUID
	Agent string
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
	repo  Repository
	tasks TaskChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks}
}

// Handle quita al dev. Solo lo hace el TL de la etapa, y solo si el dev no tiene
// tareas abiertas en ella.
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
	hasTasks, err := h.tasks.HasOpenAssigned(ctx, s.ID, cmd.Agent)
	if err != nil {
		return Result{}, err
	}
	if hasTasks {
		return Result{}, domain.ErrMemberHasTasks
	}
	if err := h.repo.RemoveMember(ctx, s.ID, cmd.Agent); err != nil {
		return Result{}, err
	}
	members := slices.DeleteFunc(slices.Clone(s.Members), func(m string) bool { return m == cmd.Agent })
	return Result{ID: s.ID, Lead: s.Lead, Members: append([]string{}, members...)}, nil
}
