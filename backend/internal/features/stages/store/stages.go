// Package store implementa la persistencia de la feature stages.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/stages/domain"
	"pmt/internal/features/stages/store/sqlcgen"
	"pmt/internal/platform/postgres"
)

// Stages implementa con PostgreSQL las interfaces que piden las slices.
type Stages struct {
	queries *sqlcgen.Queries
}

// NewStages crea el repositorio de etapas.
func NewStages(db sqlcgen.DBTX) *Stages {
	return &Stages{queries: sqlcgen.New(db)}
}

// Create guarda una etapa nueva al final de su proyecto.
func (s *Stages) Create(ctx context.Context, stage domain.Stage) (domain.Stage, error) {
	row, err := s.queries.CreateStage(ctx, sqlcgen.CreateStageParams{
		ID:          stage.ID,
		ProjectID:   stage.ProjectID,
		Name:        stage.Name,
		Description: stage.Description,
	})
	switch {
	case postgres.IsForeignKeyViolation(err):
		return domain.Stage{}, domain.ErrProjectNotFound
	case postgres.IsUniqueViolation(err):
		return domain.Stage{}, domain.ErrPositionTaken
	case err != nil:
		return domain.Stage{}, fmt.Errorf("crear etapa: %w", err)
	}
	return toDomain(row), nil
}

// Get busca una etapa por su ID.
func (s *Stages) Get(ctx context.Context, id uuid.UUID) (domain.Stage, error) {
	row, err := s.queries.GetStage(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	if err != nil {
		return domain.Stage{}, fmt.Errorf("obtener etapa: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve las etapas de un proyecto en orden.
func (s *Stages) List(ctx context.Context, projectID uuid.UUID) ([]domain.Stage, error) {
	rows, err := s.queries.ListStages(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listar etapas: %w", err)
	}
	stages := make([]domain.Stage, 0, len(rows))
	for _, row := range rows {
		stages = append(stages, toDomain(row))
	}
	return stages, nil
}

// Update guarda los cambios de una etapa.
func (s *Stages) Update(ctx context.Context, stage domain.Stage) (domain.Stage, error) {
	row, err := s.queries.UpdateStage(ctx, sqlcgen.UpdateStageParams{
		ID:          stage.ID,
		Name:        stage.Name,
		Description: stage.Description,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	if err != nil {
		return domain.Stage{}, fmt.Errorf("actualizar etapa: %w", err)
	}
	return toDomain(row), nil
}

// Delete elimina una etapa.
func (s *Stages) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.queries.DeleteStage(ctx, id)
	if postgres.IsForeignKeyViolation(err) {
		return domain.ErrStageHasTasks
	}
	if err != nil {
		return fmt.Errorf("eliminar etapa: %w", err)
	}
	if deleted == 0 {
		return domain.ErrStageNotFound
	}
	return nil
}

func toDomain(row sqlcgen.Stage) domain.Stage {
	return domain.Stage{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		Name:        row.Name,
		Description: row.Description,
		Position:    int(row.Position),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
