// Package listmembers consulta los miembros de un proyecto.
package listmembers

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Members(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
	users  domain.Users
}

// New crea el handler. users es la feature auth, que da username y nombre.
func New(reader Reader, users domain.Users) *Handler {
	return &Handler{reader: reader, users: users}
}

// Handle devuelve los miembros del proyecto: el owner, los editores y los
// viewers. Cualquier miembro, con cualquier perfil, los consulta; que quien
// actúa sea miembro lo comprueba la membresía antes de llegar aquí.
func (h *Handler) Handle(ctx context.Context, projectID uuid.UUID) ([]domain.MemberItem, error) {
	members, err := h.reader.Members(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.MemberItem, 0, len(members))
	for _, m := range members {
		user, _, err := h.users.FindByID(ctx, m.UserID)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.MemberItem{Username: user.Username, Name: user.Name, Role: m.Role})
	}
	return items, nil
}
