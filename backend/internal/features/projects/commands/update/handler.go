// Package update cambia los datos de un proyecto.
package update

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
	Update(ctx context.Context, p domain.Project) (domain.Project, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// Result es el proyecto actualizado.
type Result struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle actualiza el proyecto.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	p, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := p.Update(cmd.Name, cmd.Description); err != nil {
		return Result{}, err
	}

	updated, err := h.repo.Update(ctx, p)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
