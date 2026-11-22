// Package release libera a mano un bug tomado: el owner del proyecto lo deja sin
// tomar, con motivo, para que otra persona pueda corregirlo.
package release

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Bug, error)
	Release(ctx context.Context, b domain.Bug, holder uuid.UUID, change domain.StatusChange) (domain.Bug, error)
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

// Result es el bug liberado.
type Result struct {
	ID               uuid.UUID     `json:"id"`
	ProjectID        uuid.UUID     `json:"project_id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	StepsToReproduce string        `json:"steps_to_reproduce"`
	ProductVersion   string        `json:"product_version"`
	Status           domain.Status `json:"status"`
	ReportedBy       string        `json:"reported_by,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`

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

// Handle deja el bug sin tomar y registra el motivo en su historial de status.
// Solo PP o el usuario, y solo si es el owner del proyecto. El status no cambia y
// nadie lo toma: lo toma quien lo pase a in_dev.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanRelease(cmd.Actor); err != nil {
		return Result{}, err
	}
	b, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	owner, err := h.owners.IsOwner(ctx, b.ProjectID, cmd.Actor.UserID)
	if err != nil {
		return Result{}, err
	}
	if !owner {
		return Result{}, domain.ErrReleaseNotOwner
	}
	holder := b.HolderID
	change, err := b.Release(cmd.Actor, cmd.Reason)
	if err != nil {
		return Result{}, err
	}
	updated, err := h.repo.Release(ctx, b, holder, change)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
