// Package domain contiene los conceptos y reglas de la feature stages.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrStageNotFound indica que la etapa no existe.
	ErrStageNotFound = kernel.NotFound("etapa no encontrada")
	// ErrStageNameRequired indica que el nombre de la etapa está vacío.
	ErrStageNameRequired = kernel.Invalid("el nombre de la etapa es obligatorio")
	// ErrProjectNotFound indica que el proyecto de la etapa no existe.
	ErrProjectNotFound = kernel.NotFound("proyecto no encontrado")
	// ErrPositionTaken indica que otra operación ocupó la posición al mismo tiempo.
	ErrPositionTaken = kernel.Conflict("la posición de la etapa cambió, intenta de nuevo")
	// ErrStageHasTasks indica que la etapa no se puede eliminar porque tiene tareas.
	ErrStageHasTasks = kernel.Conflict("la etapa tiene tareas")
)

// Stage es la raíz del agregado de una etapa. Referencia a su proyecto por ID.
type Stage struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description string
	Position    int
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewStage crea una etapa válida en su estado inicial. La posición se asigna al guardarla.
func NewStage(projectID uuid.UUID, name, description string) (Stage, error) {
	s := Stage{ID: uuid.New(), ProjectID: projectID, Status: StatusPlanning}
	if err := s.Update(name, description); err != nil {
		return Stage{}, err
	}
	return s, nil
}

// Update cambia los datos editables de la etapa.
func (s *Stage) Update(name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrStageNameRequired
	}
	s.Name = name
	s.Description = strings.TrimSpace(description)
	return nil
}
