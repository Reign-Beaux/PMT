// Package store implementa la persistencia de la feature tasks.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/features/tasks/store/sqlcgen"
	"pmt/internal/platform/postgres"
)

// Tasks implementa con PostgreSQL las interfaces que piden las slices.
type Tasks struct {
	queries *sqlcgen.Queries
}

// NewTasks crea el repositorio de tareas.
func NewTasks(db sqlcgen.DBTX) *Tasks {
	return &Tasks{queries: sqlcgen.New(db)}
}

// Create guarda una tarea nueva.
func (s *Tasks) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	row, err := s.queries.CreateTask(ctx, sqlcgen.CreateTaskParams{
		ID:          task.ID,
		StageID:     task.StageID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		Blocked:     task.Blocked,
		Version:     task.Version,
	})
	if postgres.IsForeignKeyViolation(err) {
		return domain.Task{}, domain.ErrStageNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("crear tarea: %w", err)
	}
	return toDomain(row), nil
}

// Get busca una tarea por su ID.
func (s *Tasks) Get(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	row, err := s.queries.GetTask(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("obtener tarea: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve las tareas de una etapa, de la más antigua a la más reciente.
func (s *Tasks) List(ctx context.Context, stageID uuid.UUID) ([]domain.Task, error) {
	rows, err := s.queries.ListTasks(ctx, stageID)
	if err != nil {
		return nil, fmt.Errorf("listar tareas: %w", err)
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, toDomain(row))
	}
	return tasks, nil
}

// Update guarda los cambios de una tarea si sigue en expectedVersion. Si ninguna
// fila coincide, informa un conflicto: quien relea verá la versión nueva, o un
// 404 si otra operación borró la tarea.
func (s *Tasks) Update(ctx context.Context, task domain.Task, expectedVersion int64) (domain.Task, error) {
	row, err := s.queries.UpdateTask(ctx, sqlcgen.UpdateTaskParams{
		ID:              task.ID,
		Title:           task.Title,
		Description:     task.Description,
		ExpectedVersion: expectedVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskVersionConflict
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("actualizar tarea: %w", err)
	}
	return toDomain(row), nil
}

// Delete elimina una tarea.
func (s *Tasks) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.queries.DeleteTask(ctx, id)
	if err != nil {
		return fmt.Errorf("eliminar tarea: %w", err)
	}
	if deleted == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func toDomain(row sqlcgen.Task) domain.Task {
	return domain.Task{
		ID:          row.ID,
		StageID:     row.StageID,
		Title:       row.Title,
		Description: row.Description,
		Status:      domain.Status(row.Status),
		Blocked:     row.Blocked,
		Version:     row.Version,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
