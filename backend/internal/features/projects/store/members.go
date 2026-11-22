package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/projects/domain"
	"pmt/internal/features/projects/store/sqlcgen"
	"pmt/internal/platform/postgres"
)

// Members implementa con PostgreSQL los miembros de los proyectos.
type Members struct {
	queries *sqlcgen.Queries
}

// NewMembers crea el repositorio de miembros.
func NewMembers(db sqlcgen.DBTX) *Members {
	return &Members{queries: sqlcgen.New(db)}
}

// RoleOf devuelve el proyecto de un elemento, por su tipo e ID, y el rol del
// usuario en él: vacío si no es miembro. found es false si el elemento no existe.
func (s *Members) RoleOf(ctx context.Context, userID uuid.UUID, kind string, id uuid.UUID) (projectID uuid.UUID, role domain.Role, found bool, err error) {
	row, err := s.queries.RoleOf(ctx, sqlcgen.RoleOfParams{Kind: kind, ItemID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, fmt.Errorf("consultar membresía: %w", err)
	}
	return row.ProjectID, domain.Role(row.Role), true, nil
}

// Members devuelve los miembros de un proyecto: el owner, los editores y los viewers.
func (s *Members) Members(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	rows, err := s.queries.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listar miembros: %w", err)
	}
	members := make([]domain.Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, memberToDomain(row))
	}
	return members, nil
}

// Member devuelve el rol de un usuario en un proyecto, o domain.ErrMemberNotFound.
func (s *Members) Member(ctx context.Context, projectID, userID uuid.UUID) (domain.Member, error) {
	row, err := s.queries.GetProjectMember(ctx, sqlcgen.GetProjectMemberParams{ProjectID: projectID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	if err != nil {
		return domain.Member{}, fmt.Errorf("obtener miembro: %w", err)
	}
	return memberToDomain(row), nil
}

// AddMember agrega un usuario a un proyecto.
func (s *Members) AddMember(ctx context.Context, m domain.Member) error {
	err := s.queries.InsertProjectMember(ctx, sqlcgen.InsertProjectMemberParams{ProjectID: m.ProjectID, UserID: m.UserID, Role: string(m.Role)})
	if postgres.IsUniqueViolation(err) {
		return domain.ErrMemberExists
	}
	if postgres.IsForeignKeyViolation(err) {
		return domain.ErrProjectNotFound
	}
	if err != nil {
		return fmt.Errorf("agregar miembro: %w", err)
	}
	return nil
}

// UpdateMemberRole cambia el rol de un miembro que no es owner.
func (s *Members) UpdateMemberRole(ctx context.Context, m domain.Member) error {
	updated, err := s.queries.UpdateProjectMemberRole(ctx, sqlcgen.UpdateProjectMemberRoleParams{ProjectID: m.ProjectID, UserID: m.UserID, Role: string(m.Role)})
	if err != nil {
		return fmt.Errorf("cambiar rol: %w", err)
	}
	if updated == 0 {
		return domain.ErrMemberNotFound
	}
	return nil
}

// RemoveMember quita a un miembro que no es owner.
func (s *Members) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	deleted, err := s.queries.DeleteProjectMember(ctx, sqlcgen.DeleteProjectMemberParams{ProjectID: projectID, UserID: userID})
	if err != nil {
		return fmt.Errorf("quitar miembro: %w", err)
	}
	if deleted == 0 {
		return domain.ErrMemberNotFound
	}
	return nil
}

// AdoptOrphans deja al usuario como owner de los proyectos sin owner y devuelve
// cuántos adoptó.
func (s *Members) AdoptOrphans(ctx context.Context, userID uuid.UUID) (int64, error) {
	adopted, err := s.queries.AdoptOrphanProjects(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("adoptar proyectos sin owner: %w", err)
	}
	return adopted, nil
}

func memberToDomain(row sqlcgen.ProjectMember) domain.Member {
	return domain.Member{ProjectID: row.ProjectID, UserID: row.UserID, Role: domain.Role(row.Role)}
}
