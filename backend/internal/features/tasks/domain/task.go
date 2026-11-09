// Package domain contiene los conceptos y reglas de la feature tasks.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrTaskNotFound indica que la tarea no existe.
	ErrTaskNotFound = kernel.NotFound("tarea no encontrada")
	// ErrTaskTitleRequired indica que el título de la tarea está vacío.
	ErrTaskTitleRequired = kernel.Invalid("el título de la tarea es obligatorio")
	// ErrStageNotFound indica que la etapa de la tarea no existe.
	ErrStageNotFound = kernel.NotFound("etapa no encontrada")
	// ErrTaskNotTaken indica que la tarea aún no se toma o ya terminó: no admite dueño.
	ErrTaskNotTaken = kernel.Conflict("solo se asigna una tarea ya tomada y sin terminar")
	// ErrAssignForbidden indica que el perfil no puede asignar tareas.
	ErrAssignForbidden = kernel.Forbidden("tu perfil no puede asignar tareas")
	// ErrAssigneeNotMember indica que el agente no pertenece al equipo de la etapa.
	ErrAssigneeNotMember = kernel.Conflict("el agente no está en el equipo de la etapa")
)

// Status es el estado de una tarea.
type Status string

const (
	// StatusBacklog es el estado inicial de toda tarea: definida, aún no liberada.
	StatusBacklog Status = "backlog"
	// StatusReady indica que la tarea está liberada para desarrollo.
	StatusReady Status = "ready"
	// StatusInDev indica que la tarea está en desarrollo.
	StatusInDev Status = "in_dev"
	// StatusInReview indica que la tarea está en revisión de código.
	StatusInReview Status = "in_review"
	// StatusChangesRequested indica que la tarea tiene correcciones pendientes.
	StatusChangesRequested Status = "changes_requested"
	// StatusInQA indica que la tarea está en verificación de QA.
	StatusInQA Status = "in_qa"
	// StatusPendingClose indica que QA aprobó la tarea y falta el cierre.
	StatusPendingClose Status = "pending_close"
	// StatusDone indica que la tarea está verificada y cerrada.
	StatusDone Status = "done"
	// StatusCancelled indica que la tarea ya no se hará.
	StatusCancelled Status = "cancelled"
)

// Task es la raíz del agregado de una tarea. Referencia a su etapa por ID.
// DoneAt es la fecha del cambio más reciente a done; solo viene si el status
// actual es done.
type Task struct {
	ID          uuid.UUID
	StageID     uuid.UUID
	Title       string
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Assignee    string
	DoneAt      *time.Time
}

// NewTask crea una tarea válida en su estado inicial.
func NewTask(stageID uuid.UUID, title, description string) (Task, error) {
	t := Task{ID: uuid.New(), StageID: stageID, Status: StatusBacklog}
	if err := t.Update(title, description); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Assign cambia el dueño de una tarea ya tomada. Una tarea libre se toma, no se asigna.
func (t *Task) Assign(agent string) error {
	switch t.Status {
	case StatusBacklog, StatusReady, StatusDone, StatusCancelled:
		return ErrTaskNotTaken
	}
	t.Assignee = agent
	return nil
}

// Update cambia los datos editables de la tarea.
func (t *Task) Update(title, description string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrTaskTitleRequired
	}
	t.Title = title
	t.Description = strings.TrimSpace(description)
	return nil
}
