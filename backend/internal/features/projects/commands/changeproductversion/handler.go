// Package changeproductversion sube la versión de producto de un proyecto en working.
package changeproductversion

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
	ChangeProductVersion(ctx context.Context, change domain.ProductVersionChange) (domain.Project, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID             uuid.UUID    `json:"-"`
	ProductVersion string       `json:"product_version"`
	Reason         string       `json:"reason"`
	Actor          kernel.Actor `json:"-"`
}

// Result es el proyecto con su nueva versión.
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

// Handle sube la versión de producto y guarda su registro en la misma transacción.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanPlan(cmd.Actor); err != nil {
		return Result{}, err
	}
	p, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	change, err := p.ChangeProductVersion(cmd.ProductVersion, cmd.Reason, cmd.Actor)
	if err != nil {
		return Result{}, err
	}

	updated, err := h.repo.ChangeProductVersion(ctx, change)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
