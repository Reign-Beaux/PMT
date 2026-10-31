// Package changestatus cambia el status de una tarea.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	ChangeStatus(ctx context.Context, task domain.Task, change domain.StatusChange) (domain.Task, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
	Reason string        `json:"reason"`
	Actor  kernel.Actor  `json:"-"`
}

// Result es la tarea con su nuevo status y su última versión.
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

// Handle cambia el status de la tarea. Una tarea retirada no se encuentra.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	task, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	from := task.Status
	if err := task.ChangeStatus(cmd.Status); err != nil {
		return Result{}, err
	}

	statusChange := domain.NewStatusChange(task.ID, from, task.Status, cmd.Actor, cmd.Reason)
	updated, err := h.repo.ChangeStatus(ctx, task, statusChange)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
