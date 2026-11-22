// Package addmember comparte un proyecto con un usuario como editor o viewer.
package addmember

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
	AddMember(ctx context.Context, m domain.Member) error
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID uuid.UUID    `json:"-"`
	Username  string       `json:"username"`
	Role      string       `json:"role"`
	Actor     kernel.Actor `json:"-"`
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

// Handle agrega al usuario activo con ese username. Solo el owner, sin perfil
// declarado.
func (h *Handler) Handle(ctx context.Context, cmd Command) (domain.MemberItem, error) {
	if err := domain.EnsureCanManageMembers(cmd.Actor); err != nil {
		return domain.MemberItem{}, err
	}
	role, err := domain.ParseMemberRole(cmd.Role)
	if err != nil {
		return domain.MemberItem{}, err
	}
	username := strings.TrimSpace(cmd.Username)
	if username == "" {
		return domain.MemberItem{}, domain.ErrUsernameRequired
	}
	if err := domain.EnsureOwner(ctx, h.repo, cmd.ProjectID, cmd.Actor); err != nil {
		return domain.MemberItem{}, err
	}
	user, found, err := h.users.FindByUsername(ctx, username)
	if err != nil {
		return domain.MemberItem{}, err
	}
	if !found || !user.Active {
		return domain.MemberItem{}, domain.ErrMemberUserNotFound
	}
	if err := h.repo.AddMember(ctx, domain.Member{ProjectID: cmd.ProjectID, UserID: user.ID, Role: role}); err != nil {
		return domain.MemberItem{}, err
	}
	return domain.MemberItem{Username: user.Username, Name: user.Name, Role: role}, nil
}
