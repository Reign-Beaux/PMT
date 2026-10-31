// Package listchanges consulta el historial de las tareas de una etapa.
package listchanges

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, stageID uuid.UUID, taskID *uuid.UUID) ([]domain.History, error)
}

// Original es el contenido de la fila de la tarea.
type Original struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Change es un cambio de la tarea.
type Change struct {
	ID          uuid.UUID         `json:"id"`
	Version     int               `json:"version"`
	Kind        domain.ChangeKind `json:"kind"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Reason      string            `json:"reason"`
	RequestedBy string            `json:"requested_by"`
	ChangedAt   time.Time         `json:"changed_at"`
}

// Item es la evolución completa de una tarea.
type Item struct {
	TaskID   uuid.UUID `json:"task_id"`
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

// Handle devuelve la evolución de cada tarea con al menos un cambio, incluidas las
// retiradas. Con taskID, solo la de esa tarea.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID, taskID *uuid.UUID) ([]Item, error) {
	histories, err := h.reader.List(ctx, stageID, taskID)
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
				Title:       c.Title,
				Description: c.Description,
				Reason:      c.Reason,
				RequestedBy: c.RequestedBy,
				ChangedAt:   c.ChangedAt,
			})
		}
		items = append(items, Item{
			TaskID: history.TaskID,
			Original: Original{
				Title:       history.Title,
				Description: history.Description,
				CreatedAt:   history.CreatedAt,
			},
			Changes: changes,
		})
	}
	return items, nil
}
