// Package checkteam consulta el equipo de una etapa.
package checkteam

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el TL y los devs de la etapa.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) (kernel.Team, error) {
	s, err := h.reader.Get(ctx, id)
	if err != nil {
		return kernel.Team{}, err
	}
	return s.Team(), nil
}
