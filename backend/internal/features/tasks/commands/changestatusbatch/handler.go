// Package changestatusbatch cambia el status de varias tareas de una etapa, todas o ninguna.
package changestatusbatch

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

var (
	// ErrNoTasks indica que el lote viene vacío.
	ErrNoTasks = kernel.Invalid("indica al menos una tarea")
	// ErrRepeatedTask indica que el lote nombra una tarea dos veces.
	ErrRepeatedTask = kernel.Invalid("el lote repite esta tarea")
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	ChangeBatch(ctx context.Context, ids []uuid.UUID, decide func([]domain.Task) ([]domain.Task, []domain.StatusChange, error)) ([]domain.Task, error)
}

// StageChecker da el equipo de una etapa y dice qué admiten sus tareas según el
// status de la etapa y del proyecto. Lo implementa la feature stages.
type StageChecker interface {
	Team(ctx context.Context, id uuid.UUID) (kernel.Team, error)
	EnsureWorkOpen(ctx context.Context, id uuid.UUID) error
	EnsureTaskCancellable(ctx context.Context, id uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	StageID uuid.UUID     `json:"-"`
	TaskIDs []uuid.UUID   `json:"task_ids"`
	Status  domain.Status `json:"status"`
	Reason  string        `json:"reason"`
	Actor   kernel.Actor  `json:"-"`
}

// Item es una tarea con su nuevo status y su última versión.
type Item struct {
	ID          uuid.UUID     `json:"id"`
	StageID     uuid.UUID     `json:"stage_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Assignee    string        `json:"assignee,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, stages StageChecker) *Handler {
	return &Handler{repo: repo, stages: stages}
}

// Handle cambia todas las tareas o ninguna. Cada una valida por separado lo mismo
// que change_task_status y puede tener otro origen. El error conserva su tipo y
// nombra la tarea.
func (h *Handler) Handle(ctx context.Context, cmd Command) ([]Item, error) {
	if err := ensureIDs(cmd.TaskIDs); err != nil {
		return nil, err
	}
	updated, err := h.repo.ChangeBatch(ctx, cmd.TaskIDs, func(current []domain.Task) ([]domain.Task, []domain.StatusChange, error) {
		return h.decide(ctx, cmd, current)
	})
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(updated))
	for _, task := range updated {
		items = append(items, Item(task))
	}
	return items, nil
}

// decide comprueba primero el perfil de todas, para no consultar otra feature si
// ya falla, y después ámbito, motivo y reglas entre status.
func (h *Handler) decide(ctx context.Context, cmd Command, current []domain.Task) ([]domain.Task, []domain.StatusChange, error) {
	byID := make(map[uuid.UUID]domain.Task, len(current))
	for _, task := range current {
		byID[task.ID] = task
	}
	for _, id := range cmd.TaskIDs {
		task, found := byID[id]
		if !found || task.StageID != cmd.StageID {
			return nil, nil, named(id, domain.ErrTaskNotFound)
		}
		if err := task.EnsureProfile(cmd.Status, cmd.Actor); err != nil {
			return nil, nil, named(id, err)
		}
	}

	team, err := h.stages.Team(ctx, cmd.StageID)
	if err != nil {
		return nil, nil, err
	}
	tasks := make([]domain.Task, 0, len(cmd.TaskIDs))
	changes := make([]domain.StatusChange, 0, len(cmd.TaskIDs))
	for _, id := range cmd.TaskIDs {
		task := byID[id]
		from := task.Status
		if err := task.ChangeStatus(cmd.Status, cmd.Actor, cmd.Reason, team); err != nil {
			return nil, nil, named(id, err)
		}
		if err := h.ensureRules(ctx, from, task); err != nil {
			return nil, nil, named(id, err)
		}
		tasks = append(tasks, task)
		changes = append(changes, domain.NewStatusChange(id, from, task.Status, cmd.Actor, cmd.Reason))
	}
	return tasks, changes, nil
}

func (h *Handler) ensureRules(ctx context.Context, from domain.Status, task domain.Task) error {
	if domain.IsCancelOrReopen(from, task.Status) {
		return h.stages.EnsureTaskCancellable(ctx, task.StageID)
	}
	return h.stages.EnsureWorkOpen(ctx, task.StageID)
}

func ensureIDs(ids []uuid.UUID) error {
	if len(ids) == 0 {
		return ErrNoTasks
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return named(id, ErrRepeatedTask)
		}
		seen[id] = true
	}
	return nil
}

func named(id uuid.UUID, err error) error {
	return kernel.Prefix("tarea "+id.String()+": ", err)
}
