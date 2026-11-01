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
	AcceptsWork(ctx context.Context, projectID uuid.UUID) (bool, error)
	IsFrozen(ctx context.Context, projectID uuid.UUID) (bool, error)
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

// EnsureWorkOpen aplica las reglas 1, 7, 8 y 9: una tarea solo avanza con el
// proyecto en in_dev o working y la etapa en in_dev.
func (h *Handler) EnsureWorkOpen(ctx context.Context, id uuid.UUID) error {
	s, err := h.reader.Get(ctx, id)
	if err != nil {
		return err
	}
	accepts, err := h.projects.AcceptsWork(ctx, s.ProjectID)
	if err != nil {
		return err
	}
	if !accepts {
		return domain.ErrWorkProjectClosed
	}
	if s.Status != domain.StatusInDev {
		return domain.ErrWorkStageClosed
	}
	return nil
}

// EnsureTaskCancellable aplica las reglas 2, 3, 8 y 9 a cancelar y reabrir una
// tarea. Con la etapa detenida sí se puede.
func (h *Handler) EnsureTaskCancellable(ctx context.Context, id uuid.UUID) error {
	s, err := h.reader.Get(ctx, id)
	if err != nil {
		return err
	}
	frozen, err := h.projects.IsFrozen(ctx, s.ProjectID)
	if err != nil {
		return err
	}
	if frozen {
		return domain.ErrTasksFrozenByProject
	}
	if s.Status == domain.StatusDone || s.Status == domain.StatusCancelled {
		return domain.ErrTasksFrozenByStage
	}
	return nil
}
