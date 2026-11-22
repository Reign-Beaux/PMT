// Package checkworker comprueba si un nombre es un usuario que puede trabajar en
// un proyecto: ser su TL, uno de sus devs o el dueño de una tarea.
package checkworker

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

// Members es la lectura de miembros que necesita la operación.
type Members interface {
	Member(ctx context.Context, projectID, userID uuid.UUID) (domain.Member, error)
}

// Users resuelve un usuario por username. Lo implementa la feature auth.
type Users interface {
	FindByUsername(ctx context.Context, username string) (kernel.UserRef, bool, error)
}

// Handler ejecuta la operación.
type Handler struct {
	members Members
	users   Users
}

// New crea el handler.
func New(members Members, users Users) *Handler {
	return &Handler{members: members, users: users}
}

// Handle indica si el nombre es un usuario activo. Si lo es, exige que sea owner
// o editor del proyecto: un viewer o un usuario que no es miembro responde
// conflict. Si no es un usuario activo devuelve false sin error, para que quien
// pregunta lo valide como agente.
func (h *Handler) Handle(ctx context.Context, projectID uuid.UUID, username string) (bool, error) {
	user, found, err := h.users.FindByUsername(ctx, username)
	if err != nil {
		return false, err
	}
	if !found || !user.Active {
		return false, nil
	}
	member, err := h.members.Member(ctx, projectID, user.ID)
	if err != nil && !errors.Is(err, domain.ErrMemberNotFound) {
		return true, err
	}
	return true, domain.EnsureWorks(member.Role)
}
