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
	// ErrAssignLeadForbidden indica que el perfil no puede asignar el TL.
	ErrAssignLeadForbidden = kernel.Forbidden("tu perfil no puede asignar el TL de una etapa")
	// ErrAgentNotLead indica que el agente propuesto como responsable no es TL.
	ErrAgentNotLead = kernel.Invalid("el agente no tiene perfil TL")
	// ErrAgentNotDev indica que el agente propuesto para el equipo no es dev.
	ErrAgentNotDev = kernel.Invalid("el agente no tiene perfil DevFrontend ni DevBackend")
	// ErrMemberExists indica que el agente ya pertenece al equipo.
	ErrMemberExists = kernel.Conflict("el agente ya está en el equipo de la etapa")
	// ErrMemberNotFound indica que el agente no pertenece al equipo.
	ErrMemberNotFound = kernel.NotFound("el agente no está en el equipo de la etapa")
	// ErrMemberHasTasks indica que el dev aún tiene tareas abiertas en la etapa.
	ErrMemberHasTasks = kernel.Conflict("el agente tiene tareas abiertas en la etapa: reasígnalas antes")
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

	Lead    string
	Members []string
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

// Team devuelve el equipo de la etapa.
func (s Stage) Team() kernel.Team {
	return kernel.Team{Lead: s.Lead, Members: s.Members}
}

// EnsureLead devuelve un error si el actor no es el TL de la etapa.
func (s Stage) EnsureLead(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfileTL || !s.Team().IsLead(actor) {
		return kernel.ErrNotStageLead
	}
	return nil
}

// EnsureCanAssignLead devuelve un error si el perfil no puede asignar el TL.
func EnsureCanAssignLead(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfilePP && actor.Profile != kernel.ProfileUser {
		return ErrAssignLeadForbidden
	}
	return nil
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
