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
	})
	if postgres.IsForeignKeyViolation(err) {
		return domain.Task{}, domain.ErrStageNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("crear tarea: %w", err)
	}
	return toDomain(row), nil
}

// Get busca la última versión de una tarea sin retirar por su ID.
func (s *Tasks) Get(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	row, err := s.queries.GetTask(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("obtener tarea: %w", err)
	}
	return toDomain(sqlcgen.Task(row)), nil
}

// List devuelve la última versión de las tareas sin retirar de una etapa, de la más antigua a la más reciente.
func (s *Tasks) List(ctx context.Context, stageID uuid.UUID) ([]domain.Task, error) {
	rows, err := s.queries.ListTasks(ctx, stageID)
	if err != nil {
		return nil, fmt.Errorf("listar tareas: %w", err)
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, toDomain(sqlcgen.Task(row)))
	}
	return tasks, nil
}

// Update guarda los cambios de una tarea.
func (s *Tasks) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	row, err := s.queries.UpdateTask(ctx, sqlcgen.UpdateTaskParams{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("actualizar tarea: %w", err)
	}
	return toDomain(row), nil
}

// UpdateStatus guarda el status de una tarea sin retirar y devuelve su última versión.
func (s *Tasks) UpdateStatus(ctx context.Context, task domain.Task) (domain.Task, error) {
	row, err := s.queries.UpdateTaskStatus(ctx, sqlcgen.UpdateTaskStatusParams{
		ID:     task.ID,
		Status: string(task.Status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("actualizar status de tarea: %w", err)
	}
	return toDomain(sqlcgen.Task(row)), nil
}

// Delete elimina una tarea.
func (s *Tasks) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.queries.DeleteTask(ctx, id)
	if postgres.IsForeignKeyViolation(err) {
		return domain.ErrTaskHasChanges
	}
	if err != nil {
		return fmt.Errorf("eliminar tarea: %w", err)
	}
	if deleted == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

// HasChanges indica si la tarea tiene historial de cambios.
func (s *Tasks) HasChanges(ctx context.Context, id uuid.UUID) (bool, error) {
	has, err := s.queries.HasTaskChanges(ctx, id)
	if err != nil {
		return false, fmt.Errorf("consultar cambios: %w", err)
	}
	return has, nil
}

func toDomain(row sqlcgen.Task) domain.Task {
	return domain.Task{
		ID:          row.ID,
		StageID:     row.StageID,
		Title:       row.Title,
		Description: row.Description,
		Status:      domain.Status(row.Status),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
