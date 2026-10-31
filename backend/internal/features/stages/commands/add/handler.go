// Package add registra una etapa nueva como cambio, al final de un proyecto.
package add

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Add(ctx context.Context, stage domain.Stage, change domain.Change) (domain.Stage, domain.Change, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID   uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`
}

// Result es la etapa agregada y su versión.
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

// Handle agrega la etapa y registra el cambio.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	stage, err := domain.NewStage(cmd.ProjectID, cmd.Name, cmd.Description)
	if err != nil {
		return Result{}, err
	}
	change, err := domain.NewChange(stage, domain.ChangeAdd, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	created, change, err := h.repo.Add(ctx, stage, change)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ID:          created.ID,
		ProjectID:   created.ProjectID,
		Name:        created.Name,
		Description: created.Description,
		Position:    created.Position,
		Status:      created.Status,
		Version:     change.Version,
		CreatedAt:   created.CreatedAt,
		UpdatedAt:   created.UpdatedAt,
	}, nil
}
