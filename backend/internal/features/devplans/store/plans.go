// Package store implementa la persistencia de la feature devplans.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/devplans/domain"
	"pmt/internal/features/devplans/store/sqlcgen"
)

// DB es la base de datos de la feature: consultas y transacciones. pgxpool.Pool la implementa.
type DB interface {
	sqlcgen.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Plans implementa con PostgreSQL las interfaces que piden las slices.
type Plans struct {
	db DB
}

// NewPlans crea el repositorio de planes.
func NewPlans(db DB) *Plans {
	return &Plans{db: db}
}

// Create guarda un plan con su versión 1 y sus tareas.
func (s *Plans) Create(ctx context.Context, p domain.Plan) (domain.Plan, error) {
	var created domain.Plan
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		err := queries.CreateDevPlan(ctx, sqlcgen.CreateDevPlanParams{
			ID:        p.ID,
			StageID:   toUUID(p.StageID),
			BugID:     toUUID(p.BugID),
			CreatedBy: p.CreatedBy,
		})
		if err != nil {
			return fmt.Errorf("crear plan: %w", err)
		}
		if err := saveVersion(ctx, queries, p); err != nil {
			return err
		}
		if err := queries.LinkDevPlanTasks(ctx, sqlcgen.LinkDevPlanTasksParams{PlanID: p.ID, TaskIds: p.TaskIDs}); err != nil {
			return fmt.Errorf("vincular tareas: %w", err)
		}
		created, err = get(ctx, queries, p.ID)
		return err
	})
	if err != nil {
		return domain.Plan{}, err
	}
	return created, nil
}

// Get busca un plan por su ID, con su última versión y sus tareas.
func (s *Plans) Get(ctx context.Context, id uuid.UUID) (domain.Plan, error) {
	return get(ctx, sqlcgen.New(s.db), id)
}

// Revise bloquea el plan, aplica apply con lo leído tras bloquear, y guarda la
// versión nueva y sus tareas en una transacción.
func (s *Plans) Revise(ctx context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error) {
	var updated domain.Plan
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		_, err := queries.LockDevPlan(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPlanNotFound
		}
		if err != nil {
			return fmt.Errorf("bloquear plan: %w", err)
		}
		plan, err := get(ctx, queries, id)
		if err != nil {
			return err
		}
		if err := apply(&plan); err != nil {
			return err
		}
		if err := saveVersion(ctx, queries, plan); err != nil {
			return err
		}
		if err := queries.UnlinkDevPlanTasks(ctx, id); err != nil {
			return fmt.Errorf("desvincular tareas: %w", err)
		}
		if err := queries.LinkDevPlanTasks(ctx, sqlcgen.LinkDevPlanTasksParams{PlanID: id, TaskIds: plan.TaskIDs}); err != nil {
			return fmt.Errorf("vincular tareas: %w", err)
		}
		if err := queries.TouchDevPlan(ctx, id); err != nil {
			return fmt.Errorf("actualizar plan: %w", err)
		}
		updated, err = get(ctx, queries, id)
		return err
	})
	if err != nil {
		return domain.Plan{}, err
	}
	return updated, nil
}

// List devuelve los planes que cumplen el filtro indicado: etapa, tarea o bug.
func (s *Plans) List(ctx context.Context, stageID, taskID, bugID *uuid.UUID) ([]domain.Plan, error) {
	rows, err := sqlcgen.New(s.db).ListDevPlans(ctx, sqlcgen.ListDevPlansParams{
		StageID: toUUID(stageID),
		BugID:   toUUID(bugID),
		TaskID:  toUUID(taskID),
	})
	if err != nil {
		return nil, fmt.Errorf("listar planes: %w", err)
	}
	plans := make([]domain.Plan, 0, len(rows))
	for _, row := range rows {
		plans = append(plans, toDomain(sqlcgen.GetDevPlanRow(row)))
	}
	return plans, nil
}

// ListVersions devuelve todas las versiones de un plan, en orden.
func (s *Plans) ListVersions(ctx context.Context, id uuid.UUID) ([]domain.Version, error) {
	rows, err := sqlcgen.New(s.db).ListDevPlanVersions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listar versiones: %w", err)
	}
	versions := make([]domain.Version, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, domain.Version{
			Version: int(row.Version),
			Content: domain.Content{
				Scope:              row.Scope,
				TechnicalDecisions: row.TechnicalDecisions,
				ScreenStates:       row.ScreenStates,
				EdgeCases:          row.EdgeCases,
				ErrorHandling:      row.ErrorHandling,
				AcceptanceCriteria: row.AcceptanceCriteria,
			},
			Reason:    row.Reason,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt,
		})
	}
	return versions, nil
}

func get(ctx context.Context, queries *sqlcgen.Queries, id uuid.UUID) (domain.Plan, error) {
	row, err := queries.GetDevPlan(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	if err != nil {
		return domain.Plan{}, fmt.Errorf("obtener plan: %w", err)
	}
	return toDomain(row), nil
}

func saveVersion(ctx context.Context, queries *sqlcgen.Queries, p domain.Plan) error {
	content := p.Latest.Content
	err := queries.InsertDevPlanVersion(ctx, sqlcgen.InsertDevPlanVersionParams{
		PlanID:             p.ID,
		Version:            int32(p.Latest.Version),
		Scope:              content.Scope,
		TechnicalDecisions: content.TechnicalDecisions,
		ScreenStates:       content.ScreenStates,
		EdgeCases:          content.EdgeCases,
		ErrorHandling:      content.ErrorHandling,
		AcceptanceCriteria: content.AcceptanceCriteria,
		Reason:             p.Latest.Reason,
		CreatedBy:          p.Latest.CreatedBy,
	})
	if err != nil {
		return fmt.Errorf("guardar versión: %w", err)
	}
	return nil
}

func toDomain(row sqlcgen.GetDevPlanRow) domain.Plan {
	return domain.Plan{
		ID:        row.ID,
		StageID:   fromUUID(row.StageID),
		BugID:     fromUUID(row.BugID),
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Latest: domain.Version{
			Version: int(row.Version),
			Content: domain.Content{
				Scope:              row.Scope,
				TechnicalDecisions: row.TechnicalDecisions,
				ScreenStates:       row.ScreenStates,
				EdgeCases:          row.EdgeCases,
				ErrorHandling:      row.ErrorHandling,
				AcceptanceCriteria: row.AcceptanceCriteria,
			},
			Reason:    row.Reason,
			CreatedBy: row.VersionCreatedBy,
			CreatedAt: row.VersionCreatedAt,
		},
		TaskIDs: row.TaskIds,
	}
}

func toUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func fromUUID(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	parsed := uuid.UUID(id.Bytes)
	return &parsed
}
