// Package retire retira una tarea sin borrar nada.
package retire

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	Retire(ctx context.Context, change domain.Change) (domain.Change, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`
}

// Result identifica el retiro.
type Result struct {
	RetiredID uuid.UUID `json:"retired_id"`
	Version   int       `json:"version"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle registra el retiro.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	task, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	change, err := domain.NewChange(task, domain.ChangeRetire, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	saved, err := h.repo.Retire(ctx, change)
	if err != nil {
		return Result{}, err
	}
	return Result{RetiredID: saved.TaskID, Version: saved.Version}, nil
}
