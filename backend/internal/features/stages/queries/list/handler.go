// Package list consulta las etapas de un proyecto.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.Stage, error)
}

// ProjectChecker confirma que un proyecto existe. Lo implementa la feature
// projects, que responde con un error de negocio si no lo encuentra.
type ProjectChecker interface {
	Check(ctx context.Context, id uuid.UUID) error
}

// Item es una etapa de la lista.
type Item struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	AddReason      string `json:"add_reason,omitempty"`
	AddRequestedBy string `json:"add_requested_by,omitempty"`

	Lead    string   `json:"lead,omitempty"`
	Members []string `json:"members,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader   Reader
	projects ProjectChecker
}

// New crea el handler.
func New(reader Reader, projects ProjectChecker) *Handler {
	return &Handler{reader: reader, projects: projects}
}

// Handle devuelve las etapas del proyecto en orden.
func (h *Handler) Handle(ctx context.Context, projectID uuid.UUID) ([]Item, error) {
	if err := h.projects.Check(ctx, projectID); err != nil {
		return nil, err
	}

	stages, err := h.reader.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(stages))
	for _, s := range stages {
		items = append(items, Item(s))
	}
	return items, nil
}
