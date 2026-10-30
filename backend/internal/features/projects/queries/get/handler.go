// Package get consulta un proyecto.
package get

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// Result es el proyecto consultado.
type Result struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el proyecto.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) (Result, error) {
	p, err := h.reader.Get(ctx, id)
	if err != nil {
		return Result{}, err
	}
	return Result(p), nil
}
