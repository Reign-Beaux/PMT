package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/features/tasks/store/sqlcgen"
)

// DB es la base de datos de la feature: consultas y transacciones. pgxpool.Pool la implementa.
type DB interface {
	sqlcgen.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Changes guarda los cambios. Cada uno trae el contenido completo de su versión;
// la fila de tasks conserva el original.
type Changes struct {
	db DB
}

// NewChanges crea el repositorio de cambios.
func NewChanges(db DB) *Changes {
	return &Changes{db: db}
}

// Add guarda una tarea nueva y su cambio, versión 1.
func (s *Changes) Add(ctx context.Context, task domain.Task, change domain.Change) (domain.Task, domain.Change, error) {
	var created domain.Task
	var saved domain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		if created, err = NewTasks(tx).Create(ctx, task); err != nil {
			return err
		}
		saved, err = insertChange(ctx, sqlcgen.New(tx), change, 1)
		return err
	})
	if err != nil {
		return domain.Task{}, domain.Change{}, err
	}
	return created, saved, nil
}

// Revise guarda una nueva versión de la tarea, sin tocar su fila, y devuelve la última versión.
func (s *Changes) Revise(ctx context.Context, change domain.Change) (domain.Task, domain.Change, error) {
	var updated domain.Task
	var saved domain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		version, err := nextVersion(ctx, queries, change.TaskID)
		if err != nil {
			return err
		}
		if saved, err = insertChange(ctx, queries, change, version); err != nil {
			return err
		}
		updated, err = NewTasks(tx).Get(ctx, change.TaskID)
		return err
	})
	if err != nil {
		return domain.Task{}, domain.Change{}, err
	}
	return updated, saved, nil
}

// Retire registra el retiro de la tarea sin borrar nada.
func (s *Changes) Retire(ctx context.Context, change domain.Change) (domain.Change, error) {
	var saved domain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		version, err := nextVersion(ctx, queries, change.TaskID)
		if err != nil {
			return err
		}
		if saved, err = insertChange(ctx, queries, change, version); err != nil {
			return err
		}
		if err := queries.RetireTask(ctx, change.TaskID); err != nil {
			return fmt.Errorf("retirar tarea: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Change{}, err
	}
	return saved, nil
}

// List devuelve el historial de las tareas de un etapa, incluidas las retiradas.
// Si taskID no es nulo, solo el de esa tarea.
func (s *Changes) List(ctx context.Context, stageID uuid.UUID, taskID *uuid.UUID) ([]domain.History, error) {
	queries := sqlcgen.New(s.db)
	if taskID == nil {
		rows, err := queries.ListTaskChanges(ctx, stageID)
		if err != nil {
			return nil, fmt.Errorf("listar cambios: %w", err)
		}
		return toHistories(rows), nil
	}

	rows, err := queries.ListTaskChangesByTask(ctx, sqlcgen.ListTaskChangesByTaskParams{
		StageID: stageID,
		TaskID:  *taskID,
	})
	if err != nil {
		return nil, fmt.Errorf("listar cambios: %w", err)
	}
	converted := make([]sqlcgen.ListTaskChangesRow, 0, len(rows))
	for _, row := range rows {
		converted = append(converted, sqlcgen.ListTaskChangesRow(row))
	}
	return toHistories(converted), nil
}

// nextVersion bloquea la tarea y devuelve el número de su próximo cambio.
func nextVersion(ctx context.Context, queries *sqlcgen.Queries, id uuid.UUID) (int, error) {
	if _, err := queries.LockTask(ctx, id); errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrTaskNotFound
	} else if err != nil {
		return 0, fmt.Errorf("bloquear tarea: %w", err)
	}

	version, err := queries.MaxTaskChangeVersion(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("obtener versión: %w", err)
	}
	return int(version) + 1, nil
}

func insertChange(ctx context.Context, queries *sqlcgen.Queries, change domain.Change, version int) (domain.Change, error) {
	row, err := queries.CreateTaskChange(ctx, sqlcgen.CreateTaskChangeParams{
		ID:          change.ID,
		TaskID:      change.TaskID,
		Version:     int32(version),
		Kind:        string(change.Kind),
		Title:       change.Title,
		Description: change.Description,
		Reason:      change.Reason,
		RequestedBy: change.RequestedBy,
	})
	if err != nil {
		return domain.Change{}, fmt.Errorf("guardar cambio: %w", err)
	}
	return toChange(row), nil
}

func toChange(row sqlcgen.TaskChange) domain.Change {
	return domain.Change{
		ID:          row.ID,
		TaskID:      row.TaskID,
		Version:     int(row.Version),
		Kind:        domain.ChangeKind(row.Kind),
		Title:       row.Title,
		Description: row.Description,
		Reason:      row.Reason,
		RequestedBy: row.RequestedBy,
		ChangedAt:   row.ChangedAt,
	}
}

// toHistories agrupa las filas, que llegan ordenadas por tarea y versión.
func toHistories(rows []sqlcgen.ListTaskChangesRow) []domain.History {
	histories := make([]domain.History, 0, len(rows))
	for _, row := range rows {
		if last := len(histories) - 1; last < 0 || histories[last].TaskID != row.TaskChange.TaskID {
			histories = append(histories, domain.History{
				TaskID:      row.TaskChange.TaskID,
				Title:       row.OriginalTitle,
				Description: row.OriginalDescription,
				CreatedAt:   row.OriginalCreatedAt,
			})
		}
		last := len(histories) - 1
		histories[last].Changes = append(histories[last].Changes, toChange(row.TaskChange))
	}
	return histories
}
