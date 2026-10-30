// Package store implementa la persistencia del módulo work.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/work/domain"
	"pmt/internal/features/work/store/sqlcgen"
)

// Projects implementa usecase.ProjectRepository con PostgreSQL.
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
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
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
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("actualizar proyecto: %w", err)
	}
	return toDomain(row), nil
}

// Delete elimina un proyecto.
func (s *Projects) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.queries.DeleteProject(ctx, id)
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
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
