package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/features/tasks/store/sqlcgen"
)

// StatusChanges guarda los cambios de status y su registro inmutable.
type StatusChanges struct {
	db DB
}

// NewStatusChanges crea el repositorio de cambios de status.
func NewStatusChanges(db DB) *StatusChanges {
	return &StatusChanges{db: db}
}

// ChangeStatus guarda el nuevo status y su registro en una transacción. Bloquea
// la fila para que el status de origen del registro siempre sea cierto.
func (s *StatusChanges) ChangeStatus(ctx context.Context, task domain.Task, change domain.StatusChange) (domain.Task, error) {
	var updated domain.Task
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if err := ensureStatus(ctx, queries, change); err != nil {
			return err
		}
		if err := insertStatusChange(ctx, queries, change); err != nil {
			return err
		}
		var err error
		updated, err = NewTasks(tx).UpdateStatus(ctx, task)
		return err
	})
	if err != nil {
		return domain.Task{}, err
	}
	return updated, nil
}

// ChangeBatch cambia varias tareas en una transacción: todas o ninguna. Bloquea por
// orden de ID, para que dos lotes no se bloqueen entre sí, y decide con lo leído tras
// bloquear. decide recibe las tareas que existen sin retirar y devuelve cómo quedan,
// con un registro por tarea en el mismo orden.
func (s *StatusChanges) ChangeBatch(ctx context.Context, ids []uuid.UUID, decide func([]domain.Task) ([]domain.Task, []domain.StatusChange, error)) ([]domain.Task, error) {
	var updated []domain.Task
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		repo := NewTasks(tx)
		locked, err := queries.LockTasks(ctx, ids)
		if err != nil {
			return fmt.Errorf("bloquear tareas: %w", err)
		}
		current := make([]domain.Task, 0, len(locked))
		for _, id := range locked {
			task, err := repo.Get(ctx, id)
			if err != nil {
				return err
			}
			current = append(current, task)
		}
		tasks, changes, err := decide(current)
		if err != nil {
			return err
		}
		updated = make([]domain.Task, 0, len(tasks))
		for i, task := range tasks {
			if err := insertStatusChange(ctx, queries, changes[i]); err != nil {
				return err
			}
			saved, err := repo.UpdateStatus(ctx, task)
			if err != nil {
				return err
			}
			updated = append(updated, saved)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// ListStatusChanges devuelve el historial de status por orden de fecha. Incluye
// el de elementos retirados o eliminados.
func (s *StatusChanges) ListStatusChanges(ctx context.Context, id uuid.UUID) ([]domain.StatusChange, error) {
	rows, err := sqlcgen.New(s.db).ListTaskStatusChanges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listar cambios de status: %w", err)
	}
	changes := make([]domain.StatusChange, 0, len(rows))
	for _, row := range rows {
		changes = append(changes, domain.StatusChange{
			ID:           row.ID,
			TaskID:       row.TaskID,
			From:         domain.Status(row.FromStatus),
			To:           domain.Status(row.ToStatus),
			ActorName:    row.ActorName,
			ActorProfile: row.ActorProfile,
			Reason:       row.Reason,
			ChangedAt:    row.ChangedAt,
		})
	}
	return changes, nil
}

// ensureStatus bloquea la fila y comprueba que su status siga siendo el de origen.
func ensureStatus(ctx context.Context, queries *sqlcgen.Queries, change domain.StatusChange) error {
	row, err := lockTask(ctx, queries, change.TaskID)
	if err != nil {
		return err
	}
	if domain.Status(row.Status) != change.From {
		return domain.ErrTaskStatusChanged
	}
	return nil
}

func insertStatusChange(ctx context.Context, queries *sqlcgen.Queries, change domain.StatusChange) error {
	err := queries.CreateTaskStatusChange(ctx, sqlcgen.CreateTaskStatusChangeParams{
		ID:           change.ID,
		TaskID:       change.TaskID,
		FromStatus:   string(change.From),
		ToStatus:     string(change.To),
		ActorName:    change.ActorName,
		ActorProfile: change.ActorProfile,
		Reason:       change.Reason,
	})
	if err != nil {
		return fmt.Errorf("guardar cambio de status: %w", err)
	}
	return nil
}
