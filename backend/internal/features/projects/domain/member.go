package domain

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// Role es el rol de un usuario en un proyecto.
type Role string

// Roles de un proyecto. Hay un solo owner: quien lo creó.
const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// CanWrite indica si el rol escribe en el proyecto, según su perfil. El viewer
// solo lee. Solo owner y editor trabajan en un proyecto con agentes: son los
// únicos que pueden ser TL, dev o dueño de una tarea.
func (r Role) CanWrite() bool {
	return r == RoleOwner || r == RoleEditor
}

// Reaches indica si el rol deja llegar al proyecto con la credencial del actor:
// owner y editor con cualquiera; el viewer solo con la cookie del sitio web.
func (r Role) Reaches(actor kernel.Actor) bool {
	return r.CanWrite() || r == RoleViewer && actor.InBrowser()
}

// EnsureWorks exige que un usuario pueda trabajar en el proyecto (ser su TL, dev
// o dueño de una tarea): owner o editor. role vacío es que no es miembro.
func EnsureWorks(role Role) error {
	switch {
	case role.CanWrite():
		return nil
	case role == RoleViewer:
		return ErrWorkerIsViewer
	default:
		return ErrWorkerNotMember
	}
}

// ReachableRoles son los roles con los que el actor ve un proyecto según su
// credencial: con token, owner y editor; con la cookie, también viewer.
func ReachableRoles(actor kernel.Actor) []Role {
	if actor.InBrowser() {
		return []Role{RoleOwner, RoleEditor, RoleViewer}
	}
	return []Role{RoleOwner, RoleEditor}
}

// Member es un usuario con su rol en un proyecto.
type Member struct {
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Role      Role
}

// Errores de miembros.
var (
	ErrMembersForbidden   = kernel.Forbidden("solo el usuario sin perfil declarado administra los miembros de un proyecto")
	ErrNotProjectOwner    = kernel.Forbidden("solo el owner del proyecto administra sus miembros")
	ErrViewerCannotWrite  = kernel.Forbidden("eres viewer del proyecto: solo puedes leerlo")
	ErrViewerOnlyWeb      = kernel.Forbidden("eres viewer de este proyecto: solo puedes consultarlo en el sitio web")
	ErrWorkerNotMember    = kernel.Conflict("el usuario no es miembro del proyecto: compártelo como editor antes de darle trabajo")
	ErrWorkerIsViewer     = kernel.Conflict("el usuario es viewer del proyecto: hazlo editor antes de darle trabajo")
	ErrRoleInvalid        = kernel.Invalid("rol inválido: usa editor o viewer")
	ErrUsernameRequired   = kernel.Invalid("el username es obligatorio")
	ErrMemberUserNotFound = kernel.NotFound("no hay un usuario activo con ese username")
	ErrMemberExists       = kernel.Conflict("el usuario ya es miembro del proyecto")
	ErrMemberNotFound     = kernel.NotFound("el usuario no es miembro del proyecto")
	ErrOwnerUnchangeable  = kernel.Conflict("el owner del proyecto no se quita ni cambia de rol")
	ErrNoUser             = kernel.Forbidden("no hay un usuario que quede como owner del proyecto")
)

// ParseMemberRole interpreta el rol que se da a un miembro: editor o viewer. El
// owner no se asigna: es quien crea el proyecto.
func ParseMemberRole(value string) (Role, error) {
	role := Role(strings.TrimSpace(value))
	if role != RoleEditor && role != RoleViewer {
		return "", ErrRoleInvalid
	}
	return role, nil
}

// EnsureCanManageMembers exige el perfil User: los miembros se administran sin
// perfil declarado, como las cuentas. Se comprueba antes de leer la base.
func EnsureCanManageMembers(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfileUser {
		return ErrMembersForbidden
	}
	return nil
}

// EnsureManages exige que el rol de quien actúa sea owner.
func EnsureManages(role Role) error {
	if role != RoleOwner {
		return ErrNotProjectOwner
	}
	return nil
}

// EnsureChangeable rechaza quitar o cambiar al owner.
func (m Member) EnsureChangeable() error {
	if m.Role == RoleOwner {
		return ErrOwnerUnchangeable
	}
	return nil
}

// OwnerOf exige un usuario que quede como owner de un proyecto nuevo. Un token
// de agente sin dueño de la instancia no lo tiene.
func OwnerOf(actor kernel.Actor) (uuid.UUID, error) {
	if actor.UserID == uuid.Nil {
		return uuid.Nil, ErrNoUser
	}
	return actor.UserID, nil
}

// MemberReader lee el rol de un usuario en un proyecto.
type MemberReader interface {
	Member(ctx context.Context, projectID, userID uuid.UUID) (Member, error)
}

// EnsureOwner exige que quien actúa sea el owner del proyecto. Si no es miembro,
// el proyecto no existe para él.
func EnsureOwner(ctx context.Context, members MemberReader, projectID uuid.UUID, actor kernel.Actor) error {
	if actor.UserID == uuid.Nil {
		return ErrProjectNotFound
	}
	m, err := members.Member(ctx, projectID, actor.UserID)
	if errors.Is(err, ErrMemberNotFound) {
		return ErrProjectNotFound
	}
	if err != nil {
		return err
	}
	return EnsureManages(m.Role)
}

// MemberItem es un miembro como lo ve quien consulta: su username, su nombre y
// su rol.
type MemberItem struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     Role   `json:"role"`
}

// Users resuelve usuarios por username o por ID. Lo implementa la feature auth.
type Users interface {
	FindByUsername(ctx context.Context, username string) (kernel.UserRef, bool, error)
	FindByID(ctx context.Context, id uuid.UUID) (kernel.UserRef, bool, error)
}
