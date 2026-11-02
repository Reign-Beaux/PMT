// Package listids consulta los ID de las tareas de una etapa, incluidas las retiradas.
package listids

import (
	"context"

	"github.com/google/uuid"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	ListIDs(ctx context.Context, stageID uuid.UUID) ([]uuid.UUID, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los ID de las tareas de la etapa. Una etapa sin tareas, o que no
// existe, responde lista vacía.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID) ([]uuid.UUID, error) {
	return h.reader.ListIDs(ctx, stageID)
}
