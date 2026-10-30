// Package update cambia los datos de una tarea con control de concurrencia.
package update

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	Update(ctx context.Context, t domain.Task, expectedVersion int64) (domain.Task, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID              uuid.UUID `json:"-"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ExpectedVersion int64     `json:"expected_version"`
}

// Result es la tarea actualizada.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	Blocked     bool          `json:"blocked"`
	Version     int64         `json:"version"`
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

// Handle actualiza la tarea si sigue en la versión esperada.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if cmd.ExpectedVersion <= 0 {
		return Result{}, domain.ErrExpectedVersionRequired
	}

	t, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := t.Update(cmd.Title, cmd.Description); err != nil {
		return Result{}, err
	}

	updated, err := h.repo.Update(ctx, t, cmd.ExpectedVersion)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}
