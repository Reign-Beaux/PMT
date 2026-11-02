// Package list consulta todos los proyectos.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context) ([]domain.Project, error)
}

// Item es un proyecto de la lista.
type Item struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Status         domain.Status `json:"status"`
	ProductVersion string        `json:"product_version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los proyectos.
func (h *Handler) Handle(ctx context.Context) ([]Item, error) {
	projects, err := h.reader.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(projects))
	for _, p := range projects {
		items = append(items, Item(p))
	}
	return items, nil
}
