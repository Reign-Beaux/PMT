// Package checkassigned consulta si un agente tiene tareas abiertas en una etapa.
package checkassigned

import (
	"context"

	"github.com/google/uuid"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	HasOpenAssigned(ctx context.Context, stageID uuid.UUID, agent string) (bool, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle indica si el agente tiene tareas sin terminar en la etapa.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID, agent string) (bool, error) {
	return h.reader.HasOpenAssigned(ctx, stageID, agent)
}
