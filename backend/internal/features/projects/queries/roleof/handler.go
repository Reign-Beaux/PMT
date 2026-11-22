// Package roleof resuelve el proyecto de un elemento y el rol de un usuario en
// él. Es la base de la comprobación de membresía de todas las operaciones.
package roleof

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	RoleOf(ctx context.Context, userID uuid.UUID, kind string, id uuid.UUID) (projectID uuid.UUID, role domain.Role, found bool, err error)
}

// Result es el proyecto del elemento y el rol del usuario: vacío si no es
// miembro. Found es false si el elemento no existe.
type Result struct {
	ProjectID uuid.UUID
	Role      domain.Role
	Found     bool
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle resuelve el elemento. kind es project, stage, task, bug, finding,
// test_plan o dev_plan.
func (h *Handler) Handle(ctx context.Context, userID uuid.UUID, kind string, id uuid.UUID) (Result, error) {
	projectID, role, found, err := h.reader.RoleOf(ctx, userID, kind, id)
	if err != nil {
		return Result{}, err
	}
	return Result{ProjectID: projectID, Role: role, Found: found}, nil
}
