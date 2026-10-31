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
	// ErrStageHasTasks indica que la etapa no se puede eliminar porque tiene tareas, incluidas las retiradas.
	ErrStageHasTasks = kernel.Conflict("la etapa tiene tareas, activas o retiradas")
	// ErrStageHasActiveTasks indica que la etapa no se puede retirar porque tiene tareas sin retirar.
	ErrStageHasActiveTasks = kernel.Conflict("la etapa tiene tareas activas")
	// ErrStageInOwnPlanning indica que la etapa aún se planea por sí sola y no admite cambios con historial.
	ErrStageInOwnPlanning = kernel.Conflict("la etapa está en su propio planning")
	// ErrStageLeftPlanning indica que otra operación sacó a la etapa de su planning propio al mismo tiempo.
	ErrStageLeftPlanning = kernel.Conflict("la etapa ya salió de su planning propio")
)

// Stage es la raíz del agregado de una etapa. Referencia a su proyecto por ID.
// AddReason y AddRequestedBy solo tienen valor durante su planning propio: son
// el motivo y el solicitante del registro add que se escribirá al salir de él.
type Stage struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description string
	Position    int
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time

	AddReason      string
	AddRequestedBy string
}

// NewStage crea una etapa válida en su estado inicial. La posición se asigna al guardarla.
func NewStage(projectID uuid.UUID, name, description string) (Stage, error) {
	s := Stage{ID: uuid.New(), ProjectID: projectID, Status: StatusPlanning}
	if err := s.Update(name, description); err != nil {
		return Stage{}, err
	}
	return s, nil
}

// NewOwnPlanningStage crea una etapa que se planea por sí sola, sin registro add
// hasta que salga de su planning propio.
func NewOwnPlanningStage(projectID uuid.UUID, name, description, reason, requestedBy string) (Stage, error) {
	s, err := NewStage(projectID, name, description)
	if err != nil {
		return Stage{}, err
	}
	if s.AddReason, s.AddRequestedBy, err = cleanRequest(reason, requestedBy); err != nil {
		return Stage{}, err
	}
	return s, nil
}

// InOwnPlanning indica si la etapa se está planeando por sí sola. El status
// planning no basta: también lo tienen las etapas definidas en el planning del proyecto.
func (s Stage) InOwnPlanning() bool {
	return s.AddReason != ""
}

// EnsureHistoryRule devuelve un error si la etapa aún no admite cambios con historial.
func (s Stage) EnsureHistoryRule() error {
	if s.InOwnPlanning() {
		return ErrStageInOwnPlanning
	}
	return nil
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
