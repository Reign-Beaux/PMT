// Package revise registra una nueva versión de una tarea.
package revise

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
	Revise(ctx context.Context, change domain.Change, actor kernel.Actor) (domain.Task, domain.Change, error)
}

// StageChecker confirma que la etapa de una tarea admite cambios con historial.
// Lo implementa la feature stages, conectada en app.
type StageChecker interface {
	EnsureHistoryRule(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`

	Actor kernel.Actor `json:"-"`
}

// Result es la última versión de la tarea.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	Version     int           `json:"version"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, stages StageChecker) *Handler {
	return &Handler{repo: repo, stages: stages}
}

// Handle registra el cambio con la nueva versión. La fila conserva el original. Una
// tarea ya entregada vuelve a changes_requested, aunque esté congelada, y una en
// done se rechaza: lo decide el repositorio con el status que lee al bloquearla.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	task, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := h.stages.EnsureHistoryRule(ctx, task.StageID); err != nil {
		return Result{}, err
	}
	if err := task.Update(cmd.Title, cmd.Description); err != nil {
		return Result{}, err
	}
	change, err := domain.NewChange(task, domain.ChangeEdit, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	updated, change, err := h.repo.Revise(ctx, change, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ID:          updated.ID,
		StageID:     updated.StageID,
		Title:       updated.Title,
		Description: updated.Description,
		Status:      updated.Status,
		Version:     change.Version,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, nil
}
