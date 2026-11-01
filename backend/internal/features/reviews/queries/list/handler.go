// Package list consulta los hallazgos de una tarea, un bug o una etapa.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, kind domain.ItemKind, itemID uuid.UUID, pending bool) ([]domain.Finding, error)
	ListByTasks(ctx context.Context, taskIDs []uuid.UUID, pending bool) ([]domain.Finding, error)
}

// TaskLister es lo que la operación pide a la feature tasks. Incluye las tareas
// retiradas, porque sus hallazgos sobreviven.
type TaskLister interface {
	TaskIDs(ctx context.Context, stageID uuid.UUID) ([]uuid.UUID, error)
}

// Query es la entrada: StageID, o ItemKind con ItemID.
type Query struct {
	ItemKind domain.ItemKind
	ItemID   uuid.UUID
	StageID  uuid.UUID
	Pending  bool
}

// Item es un hallazgo de la lista.
type Item struct {
	ID           uuid.UUID       `json:"id"`
	ItemKind     domain.ItemKind `json:"item_kind"`
	ItemID       uuid.UUID       `json:"item_id"`
	Description  string          `json:"description"`
	Blocking     bool            `json:"blocking"`
	Status       domain.Status   `json:"status"`
	ActorName    string          `json:"actor_name"`
	ActorProfile kernel.Profile  `json:"actor_profile"`
	ClosedBy     string          `json:"closed_by,omitempty"`
	CloseReason  string          `json:"close_reason,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	ClosedAt     *time.Time      `json:"closed_at,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
	tasks  TaskLister
}

// New crea el handler.
func New(reader Reader, tasks TaskLister) *Handler {
	return &Handler{reader: reader, tasks: tasks}
}

// Handle devuelve los hallazgos. No comprueba que el elemento o la etapa existan:
// los hallazgos sobreviven a su elemento, y sin hallazgos responde lista vacía.
func (h *Handler) Handle(ctx context.Context, query Query) ([]Item, error) {
	findings, err := h.find(ctx, query)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(findings))
	for _, f := range findings {
		items = append(items, Item(f))
	}
	return items, nil
}

func (h *Handler) find(ctx context.Context, query Query) ([]domain.Finding, error) {
	byItem := query.ItemKind != "" && query.ItemID != uuid.Nil
	partial := (query.ItemKind != "") != (query.ItemID != uuid.Nil)
	byStage := query.StageID != uuid.Nil
	if partial || byItem == byStage {
		return nil, domain.ErrListModeInvalid
	}
	if byItem {
		if err := domain.EnsureKind(query.ItemKind); err != nil {
			return nil, err
		}
		return h.reader.List(ctx, query.ItemKind, query.ItemID, query.Pending)
	}
	taskIDs, err := h.tasks.TaskIDs(ctx, query.StageID)
	if err != nil {
		return nil, err
	}
	return h.reader.ListByTasks(ctx, taskIDs, query.Pending)
}
