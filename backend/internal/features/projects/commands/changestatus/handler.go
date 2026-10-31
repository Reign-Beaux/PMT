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
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle cambia el status del proyecto.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	p, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	from := p.Status
	if err := p.ChangeStatus(cmd.Status); err != nil {
		return Result{}, err
	}

	statusChange := domain.NewStatusChange(p.ID, from, p.Status, cmd.Actor, cmd.Reason)
	updated, err := h.repo.ChangeStatus(ctx, p, statusChange)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
