// Package add registra una tarea nueva como cambio, al final de una etapa.
package add

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Add(ctx context.Context, task domain.Task, change domain.Change) (domain.Task, domain.Change, error)
}

// StageChecker confirma que una etapa existe y no está retirada. Lo implementa
// la feature stages, conectada en app.
type StageChecker interface {
	Check(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	StageID     uuid.UUID `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`
}

// Result es la tarea agregada y su versión.
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

// Handle agrega la tarea y registra el cambio.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := h.stages.Check(ctx, cmd.StageID); err != nil {
		return Result{}, err
	}
	task, err := domain.NewTask(cmd.StageID, cmd.Title, cmd.Description)
	if err != nil {
		return Result{}, err
	}
	change, err := domain.NewChange(task, domain.ChangeAdd, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	created, change, err := h.repo.Add(ctx, task, change)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ID:          created.ID,
		StageID:     created.StageID,
		Title:       created.Title,
		Description: created.Description,
		Status:      created.Status,
		Version:     change.Version,
		CreatedAt:   created.CreatedAt,
		UpdatedAt:   created.UpdatedAt,
	}, nil
}
