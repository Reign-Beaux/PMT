// Package store implementa la persistencia de la feature projects.
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

// Projects implementa con PostgreSQL las interfaces que piden las slices.
type Projects struct {
	queries *sqlcgen.Queries
}

// NewProjects crea el repositorio de proyectos.
func NewProjects(db sqlcgen.DBTX) *Projects {
	return &Projects{queries: sqlcgen.New(db)}
}

// Create guarda un proyecto nuevo.
func (s *Projects) Create(ctx context.Context, p domain.Project) (domain.Project, error) {
	row, err := s.queries.CreateProject(ctx, sqlcgen.CreateProjectParams{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Status:         string(p.Status),
		ProductVersion: p.ProductVersion,
	})
	if err != nil {
		return domain.Project{}, fmt.Errorf("crear proyecto: %w", err)
	}
	return toDomain(row), nil
}

// Get busca un proyecto por su ID.
func (s *Projects) Get(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	row, err := s.queries.GetProject(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("obtener proyecto: %w", err)
	}
	return toDomain(row), nil
}

// Lock busca un proyecto y bloquea su fila hasta el fin de la transacción.
func (s *Projects) Lock(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	row, err := s.queries.LockProject(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("bloquear proyecto: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve los proyectos, del más reciente al más antiguo.
func (s *Projects) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := s.queries.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar proyectos: %w", err)
	}
	projects := make([]domain.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, toDomain(row))
	}
	return projects, nil
}

// Update guarda los cambios de un proyecto.
func (s *Projects) Update(ctx context.Context, p domain.Project) (domain.Project, error) {
	row, err := s.queries.UpdateProject(ctx, sqlcgen.UpdateProjectParams{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		ProductVersion: p.ProductVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("actualizar proyecto: %w", err)
	}
	return toDomain(row), nil
}

// UpdateStatus guarda el status de un proyecto.
func (s *Projects) UpdateStatus(ctx context.Context, p domain.Project) (domain.Project, error) {
	row, err := s.queries.UpdateProjectStatus(ctx, sqlcgen.UpdateProjectStatusParams{
		ID:     p.ID,
		Status: string(p.Status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("actualizar status del proyecto: %w", err)
	}
	return toDomain(row), nil
}

// Delete elimina un proyecto.
func (s *Projects) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.queries.DeleteProject(ctx, id)
	if postgres.IsForeignKeyViolation(err) {
		return domain.ErrProjectHasStages
	}
	if err != nil {
		return fmt.Errorf("eliminar proyecto: %w", err)
	}
	if deleted == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

func toDomain(row sqlcgen.Project) domain.Project {
	return domain.Project{
		ID:             row.ID,
		Name:           row.Name,
		Description:    row.Description,
		Status:         domain.Status(row.Status),
		ProductVersion: row.ProductVersion,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
