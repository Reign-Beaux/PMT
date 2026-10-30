package domain

import (
	"slices"

	"pmt/internal/kernel"
)

var (
	// ErrStageStatusInvalid indica que el status no es uno de los valores conocidos.
	ErrStageStatusInvalid = kernel.Invalid("status desconocido: usa planning, stand_by, in_dev, done, stopped o cancelled")
	// ErrStageStatusTransition indica que el status actual no permite pasar al pedido.
	ErrStageStatusTransition = kernel.Conflict("transición de status no permitida")
	// ErrStageTasksPending indica que la etapa no se puede cerrar porque tiene tareas sin terminar.
	ErrStageTasksPending = kernel.Conflict("la etapa no se puede cerrar: tiene tareas que no están en done")
)

// Status es el estado de una etapa.
type Status string

const (
	// StatusPlanning es el estado inicial de toda etapa: se definen sus tareas.
	StatusPlanning Status = "planning"
	// StatusStandBy indica que la etapa está lista, esperando su turno.
	StatusStandBy Status = "stand_by"
	// StatusInDev indica que sus tareas se están trabajando.
	StatusInDev Status = "in_dev"
	// StatusDone indica que la etapa está terminada y cerrada.
	StatusDone Status = "done"
	// StatusStopped indica que la etapa está detenida por una causa mayor.
	StatusStopped Status = "stopped"
	// StatusCancelled indica que la etapa ya no se hará.
	StatusCancelled Status = "cancelled"
)

// statusTransitions indica a qué estados puede pasar cada estado.
var statusTransitions = map[Status][]Status{
	StatusPlanning:  {StatusStandBy, StatusStopped, StatusCancelled},
	StatusStandBy:   {StatusPlanning, StatusInDev, StatusStopped, StatusCancelled},
	StatusInDev:     {StatusDone, StatusStopped, StatusCancelled},
	StatusDone:      {StatusInDev},
	StatusStopped:   {StatusPlanning, StatusStandBy, StatusInDev, StatusCancelled},
	StatusCancelled: {StatusPlanning},
}

// ChangeStatus pasa la etapa a otro estado si la transición está permitida.
// La regla de cierre depende de las tareas, por lo que la aplica el handler.
func (s *Stage) ChangeStatus(status Status) error {
	if _, known := statusTransitions[status]; !known {
		return ErrStageStatusInvalid
	}
	if !slices.Contains(statusTransitions[s.Status], status) {
		return ErrStageStatusTransition
	}
	s.Status = status
	return nil
}
