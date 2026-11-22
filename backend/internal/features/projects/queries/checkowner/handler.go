// Package checkowner comprueba si un usuario es el owner de un proyecto: solo él
// libera una etapa o un bug tomados.
package checkowner

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Members es la lectura de miembros que necesita la operación.
type Members interface {
	Member(ctx context.Context, projectID, userID uuid.UUID) (domain.Member, error)
}

// Handler ejecuta la operación.
type Handler struct {
	members Members
}

// New crea el handler.
func New(members Members) *Handler {
	return &Handler{members: members}
}

// Handle indica si el usuario es el owner del proyecto. Un usuario vacío o que no
// es miembro no lo es.
func (h *Handler) Handle(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	if userID == uuid.Nil {
		return false, nil
	}
	member, err := h.members.Member(ctx, projectID, userID)
	if errors.Is(err, domain.ErrMemberNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return member.Role == domain.RoleOwner, nil
}
