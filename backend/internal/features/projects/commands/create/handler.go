// Package create registra un proyecto nuevo.
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, p domain.Project) (domain.Project, error)
}

// Command contiene los datos de entrada.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Result es el proyecto creado.
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

// Handle crea el proyecto.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	p, err := domain.NewProject(cmd.Name, cmd.Description)
	if err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, p)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}
