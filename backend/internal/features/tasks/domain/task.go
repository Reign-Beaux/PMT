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
	// ErrExpectedVersionRequired indica que falta la versión esperada o no es válida.
	ErrExpectedVersionRequired = kernel.Invalid("expected_version es obligatorio y debe ser mayor que 0")
	// ErrTaskVersionConflict indica que la tarea cambió desde que se leyó.
	ErrTaskVersionConflict = kernel.Conflict("la tarea cambió desde que se leyó: vuelve a leerla antes de reintentar")
)

// Status es el estado de una tarea.
type Status string

// StatusBacklog es el estado inicial de toda tarea.
const StatusBacklog Status = "backlog"

// Task es la raíz del agregado de una tarea. Referencia a su etapa por ID.
type Task struct {
	ID          uuid.UUID
	StageID     uuid.UUID
	Title       string
	Description string
	Status      Status
	Blocked     bool
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask crea una tarea válida en su estado inicial.
func NewTask(stageID uuid.UUID, title, description string) (Task, error) {
	t := Task{ID: uuid.New(), StageID: stageID, Status: StatusBacklog, Version: 1}
	if err := t.Update(title, description); err != nil {
		return Task{}, err
	}
	return t, nil
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
