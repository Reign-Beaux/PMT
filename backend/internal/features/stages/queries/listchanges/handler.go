// Package listchanges consulta el historial de las etapas de un proyecto.
package listchanges

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, projectID uuid.UUID, stageID *uuid.UUID) ([]domain.History, error)
}

// Original es el contenido de la fila de la etapa.
type Original struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Change es un cambio de la etapa.
type Change struct {
	ID          uuid.UUID         `json:"id"`
	Version     int               `json:"version"`
	Kind        domain.ChangeKind `json:"kind"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Reason      string            `json:"reason"`
	RequestedBy string            `json:"requested_by"`
	ChangedAt   time.Time         `json:"changed_at"`
}

// Item es la evolución completa de una etapa.
type Item struct {
	StageID  uuid.UUID `json:"stage_id"`
	Original Original  `json:"original"`
	Changes  []Change  `json:"changes"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve la evolución de cada etapa con al menos un cambio, incluidas las
// retiradas. Con stageID, solo la de esa etapa.
func (h *Handler) Handle(ctx context.Context, projectID uuid.UUID, stageID *uuid.UUID) ([]Item, error) {
	histories, err := h.reader.List(ctx, projectID, stageID)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(histories))
	for _, history := range histories {
		changes := make([]Change, 0, len(history.Changes))
		for _, c := range history.Changes {
			changes = append(changes, Change{
				ID:          c.ID,
				Version:     c.Version,
				Kind:        c.Kind,
				Name:        c.Name,
				Description: c.Description,
				Reason:      c.Reason,
				RequestedBy: c.RequestedBy,
				ChangedAt:   c.ChangedAt,
			})
		}
		items = append(items, Item{
			StageID: history.StageID,
			Original: Original{
				Name:        history.Name,
				Description: history.Description,
				CreatedAt:   history.CreatedAt,
			},
			Changes: changes,
		})
	}
	return items, nil
}
