// Package checktasks consulta qué operaciones admiten las tareas de una etapa.
package checktasks

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	HasAdd(ctx context.Context, id uuid.UUID) (bool, error)
}

// ProjectChecker confirma el status de un proyecto. Lo implementa la feature
// projects, conectada en app.
type ProjectChecker interface {
	EnsurePlanning(ctx context.Context, projectID uuid.UUID) error
	EnsureWorking(ctx context.Context, projectID uuid.UUID) error
}

// Handler ejecuta la operación.
type Handler struct {
	reader   Reader
	projects ProjectChecker
}

// New crea el handler.
func New(reader Reader, projects ProjectChecker) *Handler {
	return &Handler{reader: reader, projects: projects}
}

// EnsureEditable devuelve un error de negocio si las tareas de la etapa no se
// pueden crear, editar ni eliminar directamente.
func (h *Handler) EnsureEditable(ctx context.Context, id uuid.UUID) error {
	s, err := h.reader.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.InOwnPlanning() {
		return h.projects.EnsureWorking(ctx, s.ProjectID)
	}
	if err := h.projects.EnsurePlanning(ctx, s.ProjectID); err != nil {
		return err
	}
	hasAdd, err := h.reader.HasAdd(ctx, id)
	if err != nil {
		return err
	}
	if hasAdd {
		return domain.ErrStageHasChanges
	}
	return nil
}

// EnsureHistoryRule devuelve un error de negocio si la etapa no admite cambios con historial.
func (h *Handler) EnsureHistoryRule(ctx context.Context, id uuid.UUID) error {
	s, err := h.reader.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.EnsureHistoryRule()
}
