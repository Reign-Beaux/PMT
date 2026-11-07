package domain

import (
	"strings"

	"pmt/internal/kernel"
)

var (
	planners = []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
	stoppers = []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
)

// statusProfiles indica qué perfiles pueden hacer cada transición. User tiene lo de PP.
// PC solo lee: no mueve proyectos.
var statusProfiles = map[Status]map[Status][]kernel.Profile{
	StatusPlanning:  {StatusStandBy: planners, StatusStopped: stoppers, StatusCancelled: planners},
	StatusStandBy:   {StatusPlanning: planners, StatusInDev: planners, StatusStopped: stoppers, StatusCancelled: planners},
	StatusInDev:     {StatusStopped: stoppers, StatusCancelled: planners},
	StatusWorking:   {StatusStopped: stoppers, StatusCancelled: planners},
	StatusStopped:   {StatusPlanning: stoppers, StatusStandBy: stoppers, StatusInDev: stoppers, StatusWorking: stoppers, StatusCancelled: planners},
	StatusCancelled: {StatusPlanning: planners},
}

// reasonRequired indica si la transición exige motivo.
func reasonRequired(from, to Status) bool {
	return to == StatusStopped || to == StatusCancelled || (from == StatusCancelled && to == StatusPlanning)
}

func ensureReason(from, to Status, reason string) error {
	if reasonRequired(from, to) && strings.TrimSpace(reason) == "" {
		return kernel.ErrReasonRequired
	}
	return nil
}

// ErrPlanForbidden indica que el perfil no planea proyectos.
var ErrPlanForbidden = kernel.Forbidden("solo PP o el usuario crean, editan o eliminan proyectos")

// EnsureCanPlan exige que el actor pueda crear, editar o eliminar proyectos. Se
// comprueba antes de leer o escribir en la base.
func EnsureCanPlan(actor kernel.Actor) error {
	if !actor.Profile.IsPlanner() {
		return ErrPlanForbidden
	}
	return nil
}

// EnsureCanChangeStatus rechaza, antes de leer, a un perfil que no mueve ningún proyecto,
// como PC, que solo lee.
func EnsureCanChangeStatus(actor kernel.Actor) error {
	if !kernel.HasAnyTransition(statusProfiles, actor.Profile) {
		return kernel.ErrTransitionForbidden
	}
	return nil
}
