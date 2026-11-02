// Package store implementa la persistencia de la feature bugs.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/features/bugs/store/sqlcgen"
	"pmt/internal/platform/postgres"
)

// Bugs implementa con PostgreSQL las interfaces que piden las slices.
type Bugs struct {
	queries *sqlcgen.Queries
}

// NewBugs crea el repositorio de bugs.
func NewBugs(db sqlcgen.DBTX) *Bugs {
	return &Bugs{queries: sqlcgen.New(db)}
}

// Create guarda un bug nuevo.
func (s *Bugs) Create(ctx context.Context, bug domain.Bug) (domain.Bug, error) {
	row, err := s.queries.CreateBug(ctx, sqlcgen.CreateBugParams{
		ID:               bug.ID,
		ProjectID:        bug.ProjectID,
		Title:            bug.Title,
		Description:      bug.Description,
		StepsToReproduce: bug.StepsToReproduce,
		ProductVersion:   bug.ProductVersion,
		Status:           string(bug.Status),
		ReportedBy:       pgtype.Text{String: bug.ReportedBy, Valid: bug.ReportedBy != ""},
	})
	if postgres.IsForeignKeyViolation(err) {
		return domain.Bug{}, domain.ErrProjectNotFound
	}
	if err != nil {
		return domain.Bug{}, fmt.Errorf("crear bug: %w", err)
	}
	return toDomain(row), nil
}

// Get busca un bug por su ID.
func (s *Bugs) Get(ctx context.Context, id uuid.UUID) (domain.Bug, error) {
	row, err := s.queries.GetBug(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Bug{}, domain.ErrBugNotFound
	}
	if err != nil {
		return domain.Bug{}, fmt.Errorf("obtener bug: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve los bugs de un proyecto por fecha de creación.
func (s *Bugs) List(ctx context.Context, projectID uuid.UUID) ([]domain.Bug, error) {
	rows, err := s.queries.ListBugs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listar bugs: %w", err)
	}
	bugs := make([]domain.Bug, 0, len(rows))
	for _, row := range rows {
		bugs = append(bugs, toDomain(row))
	}
	return bugs, nil
}

// Update guarda los cambios de un bug.
func (s *Bugs) Update(ctx context.Context, bug domain.Bug) (domain.Bug, error) {
	row, err := s.queries.UpdateBug(ctx, sqlcgen.UpdateBugParams{
		ID:               bug.ID,
		Title:            bug.Title,
		Description:      bug.Description,
		StepsToReproduce: bug.StepsToReproduce,
		ProductVersion:   bug.ProductVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Bug{}, domain.ErrBugNotFound
	}
	if err != nil {
		return domain.Bug{}, fmt.Errorf("actualizar bug: %w", err)
	}
	return toDomain(row), nil
}

// UpdateStatus guarda el status de un bug.
func (s *Bugs) UpdateStatus(ctx context.Context, bug domain.Bug) (domain.Bug, error) {
	row, err := s.queries.UpdateBugStatus(ctx, sqlcgen.UpdateBugStatusParams{
		ID:     bug.ID,
		Status: string(bug.Status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Bug{}, domain.ErrBugNotFound
	}
	if err != nil {
		return domain.Bug{}, fmt.Errorf("actualizar status del bug: %w", err)
	}
	return toDomain(row), nil
}

func toDomain(row sqlcgen.Bug) domain.Bug {
	return domain.Bug{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		Title:            row.Title,
		Description:      row.Description,
		StepsToReproduce: row.StepsToReproduce,
		ProductVersion:   row.ProductVersion,
		Status:           domain.Status(row.Status),
		ReportedBy:       row.ReportedBy.String,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
