// Package list consulta los planes de desarrollo de una etapa, una tarea o un bug.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/devplans/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, stageID, taskID, bugID *uuid.UUID) ([]domain.Plan, error)
}

// Query es la entrada: exactamente uno de sus filtros.
type Query struct {
	StageID *uuid.UUID
	TaskID  *uuid.UUID
	BugID   *uuid.UUID
}

// Item es un plan de la lista, con su última versión y sus tareas.
type Item struct {
	ID                 uuid.UUID   `json:"id"`
	StageID            *uuid.UUID  `json:"stage_id,omitempty"`
	BugID              *uuid.UUID  `json:"bug_id,omitempty"`
	CreatedBy          string      `json:"created_by"`
	Version            int         `json:"version"`
	Scope              string      `json:"scope"`
	TechnicalDecisions string      `json:"technical_decisions"`
	ScreenStates       string      `json:"screen_states"`
	EdgeCases          string      `json:"edge_cases"`
	ErrorHandling      string      `json:"error_handling"`
	AcceptanceCriteria string      `json:"acceptance_criteria"`
	Reason             string      `json:"reason,omitempty"`
	VersionCreatedBy   string      `json:"version_created_by"`
	TaskIDs            []uuid.UUID `json:"task_ids"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los planes. No comprueba que el elemento exista: los planes
// sobreviven a él, y uno sin planes responde lista vacía.
func (h *Handler) Handle(ctx context.Context, query Query) ([]Item, error) {
	filters := 0
	for _, id := range []*uuid.UUID{query.StageID, query.TaskID, query.BugID} {
		if id != nil {
			filters++
		}
	}
	if filters != 1 {
		return nil, domain.ErrListModeInvalid
	}

	plans, err := h.reader.List(ctx, query.StageID, query.TaskID, query.BugID)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(plans))
	for _, p := range plans {
		content := p.Latest.Content
		items = append(items, Item{
			ID:                 p.ID,
			StageID:            p.StageID,
			BugID:              p.BugID,
			CreatedBy:          p.CreatedBy,
			Version:            p.Latest.Version,
			Scope:              content.Scope,
			TechnicalDecisions: content.TechnicalDecisions,
			ScreenStates:       content.ScreenStates,
			EdgeCases:          content.EdgeCases,
			ErrorHandling:      content.ErrorHandling,
			AcceptanceCriteria: content.AcceptanceCriteria,
			Reason:             p.Latest.Reason,
			VersionCreatedBy:   p.Latest.CreatedBy,
			TaskIDs:            p.TaskIDs,
			CreatedAt:          p.CreatedAt,
			UpdatedAt:          p.UpdatedAt,
		})
	}
	return items, nil
}
