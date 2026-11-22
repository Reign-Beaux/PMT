// Package release libera a mano una etapa tomada: el owner del proyecto la deja
// sin tomar, con motivo, para que otra persona pueda trabajarla.
package release

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	Release(ctx context.Context, s domain.Stage, holder uuid.UUID, change domain.StatusChange) (domain.Stage, error)
}

// Owners indica si un usuario es el owner de un proyecto. Lo implementa la
// feature projects.
type Owners interface {
	IsOwner(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID    `json:"-"`
	Reason string       `json:"reason"`
	Actor  kernel.Actor `json:"-"`
}

// Result es la etapa liberada.
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

	kernel.Hold
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	owners Owners
}

// New crea el handler.
func New(repo Repository, owners Owners) *Handler {
	return &Handler{repo: repo, owners: owners}
}

// Handle deja la etapa sin tomar y registra el motivo en su historial de status.
// Solo PP o el usuario, y solo si es el owner del proyecto. El status no cambia y
// nadie la toma: la toman start_stage o reactivarla.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanRelease(cmd.Actor); err != nil {
		return Result{}, err
	}
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	owner, err := h.owners.IsOwner(ctx, s.ProjectID, cmd.Actor.UserID)
	if err != nil {
		return Result{}, err
	}
	if !owner {
		return Result{}, domain.ErrReleaseNotOwner
	}
	holder := s.HolderID
	change, err := s.Release(cmd.Actor, cmd.Reason)
	if err != nil {
		return Result{}, err
	}
	updated, err := h.repo.Release(ctx, s, holder, change)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
