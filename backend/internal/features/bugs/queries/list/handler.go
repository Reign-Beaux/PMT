// Package list consulta los bugs de un proyecto.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.Bug, error)
}

// Item es un bug de la lista.
type Item struct {
	ID               uuid.UUID     `json:"id"`
	ProjectID        uuid.UUID     `json:"project_id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	StepsToReproduce string        `json:"steps_to_reproduce"`
	ProductVersion   string        `json:"product_version"`
	Status           domain.Status `json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los bugs del proyecto por fecha de creación.
func (h *Handler) Handle(ctx context.Context, projectID uuid.UUID) ([]Item, error) {
	bugs, err := h.reader.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(bugs))
	for _, b := range bugs {
		items = append(items, Item(b))
	}
	return items, nil
}
