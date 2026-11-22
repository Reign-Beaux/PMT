// Package updatemember cambia el rol de un miembro de un proyecto entre editor y
// viewer.
package updatemember

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
	UpdateMemberRole(ctx context.Context, m domain.Member) error
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID uuid.UUID    `json:"-"`
	Username  string       `json:"-"`
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

// Handle cambia el rol. Solo el owner, sin perfil declarado. El owner no cambia
// de rol.
func (h *Handler) Handle(ctx context.Context, cmd Command) (domain.MemberItem, error) {
	if err := domain.EnsureCanManageMembers(cmd.Actor); err != nil {
		return domain.MemberItem{}, err
	}
	role, err := domain.ParseMemberRole(cmd.Role)
	if err != nil {
		return domain.MemberItem{}, err
	}
	if err := domain.EnsureOwner(ctx, h.repo, cmd.ProjectID, cmd.Actor); err != nil {
		return domain.MemberItem{}, err
	}
	user, found, err := h.users.FindByUsername(ctx, strings.TrimSpace(cmd.Username))
	if err != nil {
		return domain.MemberItem{}, err
	}
	if !found {
		return domain.MemberItem{}, domain.ErrMemberNotFound
	}
	member, err := h.repo.Member(ctx, cmd.ProjectID, user.ID)
	if err != nil {
		return domain.MemberItem{}, err
	}
	if err := member.EnsureChangeable(); err != nil {
		return domain.MemberItem{}, err
	}
	member.Role = role
	if err := h.repo.UpdateMemberRole(ctx, member); err != nil {
		return domain.MemberItem{}, err
	}
	return domain.MemberItem{Username: user.Username, Name: user.Name, Role: role}, nil
}
