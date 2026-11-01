// Package list consulta los planes de pruebas de una etapa.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, stageID uuid.UUID, taskID *uuid.UUID) ([]domain.Plan, error)
}

// Item es un plan de la lista, con su última versión y sus tareas.
type Item struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Status      domain.Status `json:"status"`
	CreatedBy   string        `json:"created_by"`
	ReviewNotes string        `json:"review_notes,omitempty"`
	ReviewedBy  string        `json:"reviewed_by,omitempty"`
	Version     int           `json:"version"`
	HappyPath   string        `json:"happy_path"`
	ErrorCases  string        `json:"error_cases"`
	Reason      string        `json:"reason,omitempty"`
	TaskIDs     []uuid.UUID   `json:"task_ids"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los planes. No comprueba que la etapa exista: sobreviven a ella,
// y una sin planes responde lista vacía.
func (h *Handler) Handle(ctx context.Context, stageID uuid.UUID, taskID *uuid.UUID) ([]Item, error) {
	plans, err := h.reader.List(ctx, stageID, taskID)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(plans))
	for _, p := range plans {
		items = append(items, Item{
			ID:          p.ID,
			StageID:     p.StageID,
			Title:       p.Title,
			Status:      p.Status,
			CreatedBy:   p.CreatedBy,
			ReviewNotes: p.ReviewNotes,
			ReviewedBy:  p.ReviewedBy,
			Version:     p.Latest.Version,
			HappyPath:   p.Latest.HappyPath,
			ErrorCases:  p.Latest.ErrorCases,
			Reason:      p.Latest.Reason,
			TaskIDs:     p.TaskIDs,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}
	return items, nil
}
