// Package store implementa la persistencia de la feature qa.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/qa/domain"
	"pmt/internal/features/qa/store/sqlcgen"
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
		err := queries.CreateTestPlan(ctx, sqlcgen.CreateTestPlanParams{
			ID:        p.ID,
			StageID:   p.StageID,
			Title:     p.Title,
			CreatedBy: p.CreatedBy,
		})
		if err != nil {
			return fmt.Errorf("crear plan: %w", err)
		}
		if err := saveVersion(ctx, queries, p); err != nil {
			return err
		}
		if err := queries.LinkTestPlanTasks(ctx, sqlcgen.LinkTestPlanTasksParams{PlanID: p.ID, TaskIds: p.TaskIDs}); err != nil {
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
// versión nueva, sus tareas y el status en una transacción.
func (s *Plans) Revise(ctx context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error) {
	return s.change(ctx, id, func(queries *sqlcgen.Queries, p *domain.Plan) error {
		if err := apply(p); err != nil {
			return err
		}
		if err := saveVersion(ctx, queries, *p); err != nil {
			return err
		}
		if err := queries.UnlinkTestPlanTasks(ctx, id); err != nil {
			return fmt.Errorf("desvincular tareas: %w", err)
		}
		if err := queries.LinkTestPlanTasks(ctx, sqlcgen.LinkTestPlanTasksParams{PlanID: id, TaskIds: p.TaskIDs}); err != nil {
			return fmt.Errorf("vincular tareas: %w", err)
		}
		return nil
	})
}

// Decide bloquea el plan, aplica apply con lo leído tras bloquear y guarda el
// status y la revisión del TL. Lo usan enviar, aprobar y devolver.
func (s *Plans) Decide(ctx context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error) {
	return s.change(ctx, id, func(_ *sqlcgen.Queries, p *domain.Plan) error {
		return apply(p)
	})
}

func (s *Plans) change(ctx context.Context, id uuid.UUID, apply func(*sqlcgen.Queries, *domain.Plan) error) (domain.Plan, error) {
	var updated domain.Plan
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		_, err := queries.LockTestPlan(ctx, id)
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
		if err := apply(queries, &plan); err != nil {
			return err
		}
		err = queries.UpdateTestPlanReview(ctx, sqlcgen.UpdateTestPlanReviewParams{
			ID:          id,
			Status:      string(plan.Status),
			ReviewNotes: plan.ReviewNotes,
			ReviewedBy:  pgtype.Text{String: plan.ReviewedBy, Valid: plan.ReviewedBy != ""},
		})
		if err != nil {
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

// List devuelve los planes de una etapa. Con taskID, solo los que la vinculan.
func (s *Plans) List(ctx context.Context, stageID uuid.UUID, taskID *uuid.UUID) ([]domain.Plan, error) {
	params := sqlcgen.ListTestPlansParams{StageID: stageID}
	if taskID != nil {
		params.TaskID = pgtype.UUID{Bytes: *taskID, Valid: true}
	}
	rows, err := sqlcgen.New(s.db).ListTestPlans(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("listar planes: %w", err)
	}
	plans := make([]domain.Plan, 0, len(rows))
	for _, row := range rows {
		plans = append(plans, toDomain(sqlcgen.GetTestPlanRow(row)))
	}
	return plans, nil
}

// ListVersions devuelve todas las versiones de un plan, en orden.
func (s *Plans) ListVersions(ctx context.Context, id uuid.UUID) ([]domain.Version, error) {
	rows, err := sqlcgen.New(s.db).ListTestPlanVersions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listar versiones: %w", err)
	}
	versions := make([]domain.Version, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, domain.Version{
			Version:    int(row.Version),
			HappyPath:  row.HappyPath,
			ErrorCases: row.ErrorCases,
			Reason:     row.Reason,
			CreatedBy:  row.CreatedBy,
			CreatedAt:  row.CreatedAt,
		})
	}
	return versions, nil
}

func get(ctx context.Context, queries *sqlcgen.Queries, id uuid.UUID) (domain.Plan, error) {
	row, err := queries.GetTestPlan(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	if err != nil {
		return domain.Plan{}, fmt.Errorf("obtener plan: %w", err)
	}
	return toDomain(row), nil
}

func saveVersion(ctx context.Context, queries *sqlcgen.Queries, p domain.Plan) error {
	err := queries.InsertTestPlanVersion(ctx, sqlcgen.InsertTestPlanVersionParams{
		PlanID:     p.ID,
		Version:    int32(p.Latest.Version),
		HappyPath:  p.Latest.HappyPath,
		ErrorCases: p.Latest.ErrorCases,
		Reason:     p.Latest.Reason,
		CreatedBy:  p.Latest.CreatedBy,
	})
	if err != nil {
		return fmt.Errorf("guardar versión: %w", err)
	}
	return nil
}

func toDomain(row sqlcgen.GetTestPlanRow) domain.Plan {
	return domain.Plan{
		ID:          row.ID,
		StageID:     row.StageID,
		Title:       row.Title,
		Status:      domain.Status(row.Status),
		CreatedBy:   row.CreatedBy,
		ReviewNotes: row.ReviewNotes,
		ReviewedBy:  row.ReviewedBy.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Latest: domain.Version{
			Version:    int(row.Version),
			HappyPath:  row.HappyPath,
			ErrorCases: row.ErrorCases,
			Reason:     row.Reason,
			CreatedBy:  row.VersionCreatedBy,
			CreatedAt:  row.VersionCreatedAt,
		},
		TaskIDs: row.TaskIds,
	}
}
