// Package revise registra una nueva versión de una etapa.
package revise

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	Revise(ctx context.Context, change domain.Change) (domain.Stage, domain.Change, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID          uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`

	Actor kernel.Actor `json:"-"`
}

// Result es la última versión de la etapa.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	Version     int           `json:"version"`
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

// Handle registra el cambio con la nueva versión. La fila conserva el original.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanRevise(cmd.Actor); err != nil {
		return Result{}, err
	}
	stage, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := stage.EnsureReviser(cmd.Actor); err != nil {
		return Result{}, err
	}
	if err := stage.EnsureHistoryRule(); err != nil {
		return Result{}, err
	}
	if err := stage.Update(cmd.Name, cmd.Description); err != nil {
		return Result{}, err
	}
	change, err := domain.NewChange(stage, domain.ChangeEdit, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	updated, change, err := h.repo.Revise(ctx, change)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ID:          updated.ID,
		ProjectID:   updated.ProjectID,
		Name:        updated.Name,
		Description: updated.Description,
		Position:    updated.Position,
		Status:      updated.Status,
		Version:     change.Version,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, nil
}
