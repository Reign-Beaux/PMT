// Package list consulta los agentes.
package list

import (
	"context"
	"time"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context) ([]domain.Agent, error)
}

// Item es un agente, sin datos de su token.
type Item struct {
	Name      string
	Profile   kernel.Profile
	Revoked   bool
	CreatedAt time.Time
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los agentes por nombre.
func (h *Handler) Handle(ctx context.Context) ([]Item, error) {
	agents, err := h.reader.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(agents))
	for _, agent := range agents {
		items = append(items, Item{
			Name:      agent.Name,
			Profile:   agent.Profile,
			Revoked:   agent.Revoked(),
			CreatedAt: agent.CreatedAt,
		})
	}
	return items, nil
}
