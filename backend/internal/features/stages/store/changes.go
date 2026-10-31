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

// DB es la base de datos de la feature: consultas y transacciones. pgxpool.Pool la implementa.
type DB interface {
	sqlcgen.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Changes guarda los cambios. Cada uno trae el contenido completo de su versión;
// la fila de stages conserva el original.
type Changes struct {
	db DB
}

// NewChanges crea el repositorio de cambios.
func NewChanges(db DB) *Changes {
	return &Changes{db: db}
}

// Add guarda una etapa nueva y su cambio, versión 1.
func (s *Changes) Add(ctx context.Context, stage domain.Stage, change domain.Change) (domain.Stage, domain.Change, error) {
	var created domain.Stage
	var saved domain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		if created, err = NewStages(tx).Create(ctx, stage); err != nil {
			return err
		}
		saved, err = insertChange(ctx, sqlcgen.New(tx), change, 1)
		return err
	})
	if postgres.IsUniqueViolation(err) {
		return domain.Stage{}, domain.Change{}, domain.ErrPositionTaken
	}
	if err != nil {
		return domain.Stage{}, domain.Change{}, err
	}
	return created, saved, nil
}

// Revise guarda una nueva versión de la etapa, sin tocar su fila, y devuelve la última versión.
func (s *Changes) Revise(ctx context.Context, change domain.Change) (domain.Stage, domain.Change, error) {
	var updated domain.Stage
	var saved domain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		version, err := nextVersion(ctx, queries, change.StageID)
		if err != nil {
			return err
		}
		if saved, err = insertChange(ctx, queries, change, version); err != nil {
			return err
		}
		updated, err = NewStages(tx).Get(ctx, change.StageID)
		return err
	})
	if err != nil {
		return domain.Stage{}, domain.Change{}, err
	}
	return updated, saved, nil
}

// Retire registra el retiro de la etapa sin borrar nada.
func (s *Changes) Retire(ctx context.Context, change domain.Change) (domain.Change, error) {
	var saved domain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		version, err := nextVersion(ctx, queries, change.StageID)
		if err != nil {
			return err
		}
		if saved, err = insertChange(ctx, queries, change, version); err != nil {
			return err
		}
		if err := queries.RetireStage(ctx, change.StageID); err != nil {
			return fmt.Errorf("retirar etapa: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Change{}, err
	}
	return saved, nil
}

// LeavePlanning saca a la etapa de su planning propio: escribe su registro add,
// versión 1 y su registro de status, guarda el nuevo status y limpia los datos pendientes. Dos salidas
// simultáneas chocan en UNIQUE (stage_id, version).
func (s *Changes) LeavePlanning(ctx context.Context, stage domain.Stage, change domain.Change, statusChange domain.StatusChange) (domain.Stage, error) {
	var updated domain.Stage
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if err := ensureStatus(ctx, queries, statusChange); err != nil {
			return err
		}
		if _, err := insertChange(ctx, queries, change, 1); err != nil {
			return err
		}
		if err := insertStatusChange(ctx, queries, statusChange); err != nil {
			return err
		}
		err := queries.LeaveStagePlanning(ctx, sqlcgen.LeaveStagePlanningParams{
			ID:     stage.ID,
			Status: string(stage.Status),
		})
		if err != nil {
			return fmt.Errorf("salir de planning propio: %w", err)
		}
		updated, err = NewStages(tx).Get(ctx, stage.ID)
		return err
	})
	if postgres.IsUniqueViolation(err) {
		return domain.Stage{}, domain.ErrStageLeftPlanning
	}
	if err != nil {
		return domain.Stage{}, err
	}
	return updated, nil
}

// List devuelve el historial de las etapas de un proyecto, incluidas las retiradas.
// Si stageID no es nulo, solo el de esa etapa.
func (s *Changes) List(ctx context.Context, projectID uuid.UUID, stageID *uuid.UUID) ([]domain.History, error) {
	queries := sqlcgen.New(s.db)
	if stageID == nil {
		rows, err := queries.ListStageChanges(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("listar cambios: %w", err)
		}
		return toHistories(rows), nil
	}

	rows, err := queries.ListStageChangesByStage(ctx, sqlcgen.ListStageChangesByStageParams{
		ProjectID: projectID,
		StageID:   *stageID,
	})
	if err != nil {
		return nil, fmt.Errorf("listar cambios: %w", err)
	}
	converted := make([]sqlcgen.ListStageChangesRow, 0, len(rows))
	for _, row := range rows {
		converted = append(converted, sqlcgen.ListStageChangesRow(row))
	}
	return toHistories(converted), nil
}

// nextVersion bloquea la etapa y devuelve el número de su próximo cambio.
func nextVersion(ctx context.Context, queries *sqlcgen.Queries, id uuid.UUID) (int, error) {
	if _, err := queries.LockStage(ctx, id); errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrStageNotFound
	} else if err != nil {
		return 0, fmt.Errorf("bloquear etapa: %w", err)
	}

	version, err := queries.MaxStageChangeVersion(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("obtener versión: %w", err)
	}
	return int(version) + 1, nil
}

func insertChange(ctx context.Context, queries *sqlcgen.Queries, change domain.Change, version int) (domain.Change, error) {
	row, err := queries.CreateStageChange(ctx, sqlcgen.CreateStageChangeParams{
		ID:          change.ID,
		StageID:     change.StageID,
		Version:     int32(version),
		Kind:        string(change.Kind),
		Name:        change.Name,
		Description: change.Description,
		Reason:      change.Reason,
		RequestedBy: change.RequestedBy,
	})
	if err != nil {
		return domain.Change{}, fmt.Errorf("guardar cambio: %w", err)
	}
	return toChange(row), nil
}

func toChange(row sqlcgen.StageChange) domain.Change {
	return domain.Change{
		ID:          row.ID,
		StageID:     row.StageID,
		Version:     int(row.Version),
		Kind:        domain.ChangeKind(row.Kind),
		Name:        row.Name,
		Description: row.Description,
		Reason:      row.Reason,
		RequestedBy: row.RequestedBy,
		ChangedAt:   row.ChangedAt,
	}
}

// toHistories agrupa las filas, que llegan ordenadas por etapa y versión.
func toHistories(rows []sqlcgen.ListStageChangesRow) []domain.History {
	histories := make([]domain.History, 0, len(rows))
	for _, row := range rows {
		if last := len(histories) - 1; last < 0 || histories[last].StageID != row.StageChange.StageID {
			histories = append(histories, domain.History{
				StageID:     row.StageChange.StageID,
				Name:        row.OriginalName,
				Description: row.OriginalDescription,
				CreatedAt:   row.OriginalCreatedAt,
			})
		}
		last := len(histories) - 1
		histories[last].Changes = append(histories[last].Changes, toChange(row.StageChange))
	}
	return histories
}
