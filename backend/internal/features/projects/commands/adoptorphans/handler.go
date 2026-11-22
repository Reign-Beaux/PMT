// Package adoptorphans deja al dueño de la instancia como owner de los proyectos
// sin owner: los que existían antes de los miembros, cuando la migración no
// encontró usuarios para asignarlos.
package adoptorphans

import (
	"context"

	"github.com/google/uuid"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	AdoptOrphans(ctx context.Context, userID uuid.UUID) (int64, error)
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle adopta los proyectos sin owner y devuelve cuántos. Sin dueño de la
// instancia no hace nada.
func (h *Handler) Handle(ctx context.Context, owner uuid.UUID) (int64, error) {
	if owner == uuid.Nil {
		return 0, nil
	}
	return h.repo.AdoptOrphans(ctx, owner)
}
