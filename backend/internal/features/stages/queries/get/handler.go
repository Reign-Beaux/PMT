// Package get consulta una etapa.
package get

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
}

// Result es la etapa consultada.
type Result struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Position    int       `json:"position"`
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

// Handle devuelve la etapa.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) (Result, error) {
	s, err := h.reader.Get(ctx, id)
	if err != nil {
		return Result{}, err
	}
	return Result(s), nil
}
