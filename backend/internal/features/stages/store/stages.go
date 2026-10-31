// Package store implementa la persistencia de la feature stages.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
		ID:             stage.ID,
		ProjectID:      stage.ProjectID,
		Name:           stage.Name,
		Description:    stage.Description,
		Status:         string(stage.Status),
		AddReason:      stage.AddReason,
		AddRequestedBy: stage.AddRequestedBy,
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

// Get busca la última versión de una etapa sin retirar por su ID.
func (s *Stages) Get(ctx context.Context, id uuid.UUID) (domain.Stage, error) {
	row, err := s.queries.GetStage(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	if err != nil {
		return domain.Stage{}, fmt.Errorf("obtener etapa: %w", err)
	}
	stage := toDomain(sqlcgen.Stage(row))
	if stage.Members, err = s.queries.ListStageMembers(ctx, id); err != nil {
		return domain.Stage{}, fmt.Errorf("listar equipo de etapa: %w", err)
	}
	return stage, nil
}

// List devuelve la última versión de las etapas sin retirar de un proyecto en orden.
func (s *Stages) List(ctx context.Context, projectID uuid.UUID) ([]domain.Stage, error) {
	rows, err := s.queries.ListStages(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listar etapas: %w", err)
	}
	members, err := s.queries.ListProjectStageMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listar equipos de etapas: %w", err)
	}
	byStage := map[uuid.UUID][]string{}
	for _, member := range members {
		byStage[member.StageID] = append(byStage[member.StageID], member.Agent)
	}
	stages := make([]domain.Stage, 0, len(rows))
	for _, row := range rows {
		stage := toDomain(sqlcgen.Stage(row))
		stage.Members = byStage[stage.ID]
		stages = append(stages, stage)
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

// UpdateStatus guarda el status de una etapa y devuelve su última versión.
func (s *Stages) UpdateStatus(ctx context.Context, stage domain.Stage) (domain.Stage, error) {
	row, err := s.queries.UpdateStageStatus(ctx, sqlcgen.UpdateStageStatusParams{
		ID:     stage.ID,
		Status: string(stage.Status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	if err != nil {
		return domain.Stage{}, fmt.Errorf("actualizar status de etapa: %w", err)
	}
	return toDomain(sqlcgen.Stage(row)), nil
}

// UpdateLead guarda el TL de una etapa sin retirar.
func (s *Stages) UpdateLead(ctx context.Context, id uuid.UUID, lead string) error {
	updated, err := s.queries.UpdateStageLead(ctx, sqlcgen.UpdateStageLeadParams{
		ID:   id,
		Lead: pgtype.Text{String: lead, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("actualizar TL de etapa: %w", err)
	}
	if updated == 0 {
		return domain.ErrStageNotFound
	}
	return nil
}

// AddMember añade un dev al equipo de una etapa.
func (s *Stages) AddMember(ctx context.Context, id uuid.UUID, agent, addedBy string) error {
	err := s.queries.AddStageMember(ctx, sqlcgen.AddStageMemberParams{StageID: id, Agent: agent, AddedBy: addedBy})
	if postgres.IsUniqueViolation(err) {
		return domain.ErrMemberExists
	}
	if err != nil {
		return fmt.Errorf("añadir miembro de etapa: %w", err)
	}
	return nil
}

// RemoveMember quita un dev del equipo de una etapa.
func (s *Stages) RemoveMember(ctx context.Context, id uuid.UUID, agent string) error {
	removed, err := s.queries.RemoveStageMember(ctx, sqlcgen.RemoveStageMemberParams{StageID: id, Agent: agent})
	if err != nil {
		return fmt.Errorf("quitar miembro de etapa: %w", err)
	}
	if removed == 0 {
		return domain.ErrMemberNotFound
	}
	return nil
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

// HasChanges indica si la etapa tiene historial de cambios.
func (s *Stages) HasChanges(ctx context.Context, id uuid.UUID) (bool, error) {
	has, err := s.queries.HasStageChanges(ctx, id)
	if err != nil {
		return false, fmt.Errorf("consultar cambios: %w", err)
	}
	return has, nil
}

// HasAdd indica si la etapa tiene un registro add, es decir, si nació con add_stage.
func (s *Stages) HasAdd(ctx context.Context, id uuid.UUID) (bool, error) {
	has, err := s.queries.HasStageAdd(ctx, id)
	if err != nil {
		return false, fmt.Errorf("consultar registro add: %w", err)
	}
	return has, nil
}

func toDomain(row sqlcgen.Stage) domain.Stage {
	return domain.Stage{
		ID:             row.ID,
		ProjectID:      row.ProjectID,
		Name:           row.Name,
		Description:    row.Description,
		Position:       int(row.Position),
		Status:         domain.Status(row.Status),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		AddReason:      row.AddReason.String,
		AddRequestedBy: row.AddRequestedBy.String,
		Lead:           row.Lead.String,
	}
}
