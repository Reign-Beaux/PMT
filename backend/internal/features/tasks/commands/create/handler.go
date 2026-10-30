// Package create registra una tarea nueva en una etapa.
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, t domain.Task) (domain.Task, error)
}

// Command contiene los datos de entrada.
type Command struct {
	StageID     uuid.UUID `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

// Result es la tarea creada.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle crea la tarea.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	t, err := domain.NewTask(cmd.StageID, cmd.Title, cmd.Description)
	if err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, t)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}
