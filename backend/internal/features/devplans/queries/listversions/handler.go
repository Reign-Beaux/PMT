// Package listversions consulta todas las versiones de un plan de desarrollo.
package listversions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/devplans/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	ListVersions(ctx context.Context, id uuid.UUID) ([]domain.Version, error)
}

// Item es una versión de la lista.
type Item struct {
	Version            int       `json:"version"`
	Scope              string    `json:"scope"`
	TechnicalDecisions string    `json:"technical_decisions"`
	ScreenStates       string    `json:"screen_states"`
	EdgeCases          string    `json:"edge_cases"`
	ErrorHandling      string    `json:"error_handling"`
	AcceptanceCriteria string    `json:"acceptance_criteria"`
	Reason             string    `json:"reason,omitempty"`
	CreatedBy          string    `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve las versiones en orden. Un ID sin versiones responde lista vacía.
func (h *Handler) Handle(ctx context.Context, id uuid.UUID) ([]Item, error) {
	versions, err := h.reader.ListVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(versions))
	for _, v := range versions {
		items = append(items, Item{
			Version:            v.Version,
			Scope:              v.Content.Scope,
			TechnicalDecisions: v.Content.TechnicalDecisions,
			ScreenStates:       v.Content.ScreenStates,
			EdgeCases:          v.Content.EdgeCases,
			ErrorHandling:      v.Content.ErrorHandling,
			AcceptanceCriteria: v.Content.AcceptanceCriteria,
			Reason:             v.Reason,
			CreatedBy:          v.CreatedBy,
			CreatedAt:          v.CreatedAt,
		})
	}
	return items, nil
}
