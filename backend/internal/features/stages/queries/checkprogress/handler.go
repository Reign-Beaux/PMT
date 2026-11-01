// Package checkprogress consulta si las etapas de un proyecto permiten cambiar su
// status. Responde con los errores de negocio de stages/domain.
package checkprogress

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

// Reader es la lectura que necesita la operación. No devuelve lo retirado.
type Reader interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.Stage, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// EnsureReleasable aplica la regla 11 a planning → stand_by.
func (h *Handler) EnsureReleasable(ctx context.Context, projectID uuid.UUID) error {
	return h.check(ctx, projectID, domain.EnsureReleasable)
}

// EnsureStartable aplica la regla 12 a stand_by → in_dev.
func (h *Handler) EnsureStartable(ctx context.Context, projectID uuid.UUID) error {
	return h.check(ctx, projectID, domain.EnsureStartable)
}

// EnsureClosed aplica las reglas 13 y 14 a la llegada a working.
func (h *Handler) EnsureClosed(ctx context.Context, projectID uuid.UUID) error {
	return h.check(ctx, projectID, domain.EnsureClosed)
}

func (h *Handler) check(ctx context.Context, projectID uuid.UUID, rule func([]domain.Stage) error) error {
	stages, err := h.reader.List(ctx, projectID)
	if err != nil {
		return err
	}
	return rule(stages)
}
