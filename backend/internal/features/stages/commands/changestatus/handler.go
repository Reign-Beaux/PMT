// Package changestatus cambia el status de una etapa.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	UpdateStatus(ctx context.Context, s domain.Stage) (domain.Stage, error)
}

// TaskChecker indica si todas las tareas de una etapa están terminadas.
// Lo implementa la feature tasks.
type TaskChecker interface {
	AllDone(ctx context.Context, stageID uuid.UUID) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
}

// Result es la etapa con su nuevo status.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo  Repository
	tasks TaskChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks}
}

// Handle cambia el status de la etapa. Solo la cierra si todas sus tareas están terminadas.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := s.ChangeStatus(cmd.Status); err != nil {
		return Result{}, err
	}
	if s.Status == domain.StatusDone {
		allDone, err := h.tasks.AllDone(ctx, s.ID)
		if err != nil {
			return Result{}, err
		}
		if !allDone {
			return Result{}, domain.ErrStageTasksPending
		}
	}

	updated, err := h.repo.UpdateStatus(ctx, s)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
