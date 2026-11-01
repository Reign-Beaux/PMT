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
	// ErrStageProjectFrozen aplica la regla 10: el proyecto congela sus etapas.
	ErrStageProjectFrozen = kernel.Conflict("el proyecto está detenido o cancelado: la etapa solo puede detenerse o cancelarse")
	// ErrStageProjectNotStarted aplica la regla 5: el proyecto aún no admite trabajo.
	ErrStageProjectNotStarted = kernel.Conflict("la etapa solo pasa a in_dev con el proyecto en in_dev o working")

	// ErrStageNotStandBy indica que start_stage solo inicia etapas en stand_by.
	ErrStageNotStandBy = kernel.Conflict("la etapa no está en stand_by")
	// ErrStageProjectNotReady indica que el proyecto aún no se libera, o que está detenido.
	ErrStageProjectNotReady = kernel.Conflict("la etapa solo se inicia con el proyecto en stand_by, in_dev o working")

	// ErrWorkProjectClosed aplica la regla 1: el proyecto no admite que la tarea avance.
	ErrWorkProjectClosed = kernel.Conflict("la tarea no avanza: el proyecto debe estar en in_dev o working")
	// ErrWorkStageClosed aplica las reglas 1, 7 y 8: la etapa no admite que la tarea avance.
	ErrWorkStageClosed = kernel.Conflict("la tarea no avanza: la etapa debe estar en in_dev")
	// ErrTasksFrozenByProject aplica la regla 9 a cancelar y reabrir tareas.
	ErrTasksFrozenByProject = kernel.Conflict("el proyecto está detenido o cancelado: sus tareas no se cancelan ni se reabren")
	// ErrTasksFrozenByStage aplica las reglas 2, 3 y 8.
	ErrTasksFrozenByStage = kernel.Conflict("la etapa está en done o cancelada: sus tareas no se cancelan ni se reabren")

	// ErrNoActiveStages aplica la regla 11.
	ErrNoActiveStages = kernel.Conflict("el proyecto no tiene etapas activas")
	// ErrStagesInPlanning aplica la regla 11.
	ErrStagesInPlanning = kernel.Conflict("el proyecto tiene etapas en planning")
	// ErrNoStagesStandBy aplica la regla 12.
	ErrNoStagesStandBy = kernel.Conflict("el proyecto no tiene etapas en stand_by")
	// ErrNoStagesDone aplica la regla 13.
	ErrNoStagesDone = kernel.Conflict("la versión 1 no tiene etapas en done")
	// ErrStagesOpen aplica las reglas 13 y 14.
	ErrStagesOpen = kernel.Conflict("la versión 1 tiene etapas sin terminar: todas deben estar en done o cancelled")
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

// EnsureReleasable aplica la regla 11: al menos una etapa activa, ninguna en planning.
func EnsureReleasable(stages []Stage) error {
	active := 0
	for _, s := range stages {
		if s.Status == StatusCancelled {
			continue
		}
		if s.Status == StatusPlanning {
			return ErrStagesInPlanning
		}
		active++
	}
	if active == 0 {
		return ErrNoActiveStages
	}
	return nil
}

// EnsureStartable aplica la regla 12: al menos una etapa en stand_by.
func EnsureStartable(stages []Stage) error {
	for _, s := range stages {
		if s.Status == StatusStandBy {
			return nil
		}
	}
	return ErrNoStagesStandBy
}

// EnsureHasActive comprueba que quede al menos una etapa sin cancelar.
func EnsureHasActive(stages []Stage) error {
	for _, s := range stages {
		if s.Status != StatusCancelled {
			return nil
		}
	}
	return ErrNoActiveStages
}

// EnsureClosed aplica las reglas 13 y 14: todas en done o cancelled, con al menos
// una en done. Una etapa detenida cuenta como sin terminar.
func EnsureClosed(stages []Stage) error {
	done := false
	for _, s := range stages {
		switch s.Status {
		case StatusDone:
			done = true
		case StatusCancelled:
		default:
			return ErrStagesOpen
		}
	}
	if !done {
		return ErrNoStagesDone
	}
	return nil
}

// EndsVersionOne indica si, con la etapa ya en su nuevo status, termina la versión 1.
// versionOne son las etapas de esa versión tras el cambio. Cancelar solo cuenta si la
// etapa es de la versión 1: cancelar una agregada no termina nada.
func EndsVersionOne(stage Stage, versionOne []Stage) bool {
	if stage.Status == StatusCancelled && !slices.ContainsFunc(versionOne, func(s Stage) bool { return s.ID == stage.ID }) {
		return false
	}
	return EnsureClosed(versionOne) == nil
}

// ChangeStatus pasa la etapa a otro estado si la transición está permitida. Las
// reglas que dependen del proyecto o de las tareas las aplica el handler.
func (s *Stage) ChangeStatus(status Status, actor kernel.Actor, reason string) error {
	if _, known := statusTransitions[status]; !known {
		return ErrStageStatusInvalid
	}
	if !slices.Contains(statusTransitions[s.Status], status) {
		return ErrStageStatusTransition
	}
	if err := s.ensureAllowed(status, actor, reason); err != nil {
		return err
	}
	s.Status = status
	return nil
}
