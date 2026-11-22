// Package removemember deja de compartir un proyecto con un miembro.
package removemember

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	domain.MemberReader
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID uuid.UUID
	Username  string
	Actor     kernel.Actor
}

// Handler ejecuta la operación.
type Handler struct {
	repo  Repository
	users domain.Users
}

// New crea el handler. users es la feature auth.
func New(repo Repository, users domain.Users) *Handler {
	return &Handler{repo: repo, users: users}
}

// Handle quita al miembro. Solo el owner, sin perfil declarado. El owner no se
// quita.
func (h *Handler) Handle(ctx context.Context, cmd Command) error {
	if err := domain.EnsureCanManageMembers(cmd.Actor); err != nil {
		return err
	}
	if err := domain.EnsureOwner(ctx, h.repo, cmd.ProjectID, cmd.Actor); err != nil {
		return err
	}
	user, found, err := h.users.FindByUsername(ctx, strings.TrimSpace(cmd.Username))
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrMemberNotFound
	}
	member, err := h.repo.Member(ctx, cmd.ProjectID, user.ID)
	if err != nil {
		return err
	}
	if err := member.EnsureChangeable(); err != nil {
		return err
	}
	return h.repo.RemoveMember(ctx, cmd.ProjectID, user.ID)
}
