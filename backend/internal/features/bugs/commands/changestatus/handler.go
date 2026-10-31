// Package changestatus cambia el status de un bug.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Bug, error)
	UpdateStatus(ctx context.Context, b domain.Bug) (domain.Bug, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
}

// Result es el bug con su nuevo status.
type Result struct {
	ID               uuid.UUID     `json:"id"`
	ProjectID        uuid.UUID     `json:"project_id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	StepsToReproduce string        `json:"steps_to_reproduce"`
	ProductVersion   string        `json:"product_version"`
	Status           domain.Status `json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle cambia el status del bug.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	bug, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := bug.ChangeStatus(cmd.Status); err != nil {
		return Result{}, err
	}

	updated, err := h.repo.UpdateStatus(ctx, bug)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
