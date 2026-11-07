package domain

import (
	"slices"
	"strings"

	"pmt/internal/kernel"
)

var (
	planners = []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
	leads    = []kernel.Profile{kernel.ProfileTL}
	reopens  = []kernel.Profile{kernel.ProfilePP, kernel.ProfileTL, kernel.ProfileUser}
	stoppers = []kernel.Profile{kernel.ProfilePP, kernel.ProfileTL, kernel.ProfileUser}
)

// statusProfiles indica qué perfiles pueden hacer cada transición. User tiene lo de
// PP. Un TL además debe ser el de la etapa. PC solo lee: no mueve etapas.
var statusProfiles = map[Status]map[Status][]kernel.Profile{
	StatusPlanning:  {StatusStandBy: planners, StatusStopped: stoppers, StatusCancelled: planners},
	StatusStandBy:   {StatusPlanning: planners, StatusInDev: leads, StatusStopped: stoppers, StatusCancelled: planners},
	StatusInDev:     {StatusDone: leads, StatusStopped: stoppers, StatusCancelled: planners},
	StatusDone:      {StatusInDev: reopens},
	StatusStopped:   {StatusPlanning: stoppers, StatusStandBy: stoppers, StatusInDev: stoppers, StatusCancelled: planners},
	StatusCancelled: {StatusPlanning: planners},
}

// reasonRequired indica si la transición exige motivo.
func reasonRequired(from, to Status) bool {
	return to == StatusStopped || to == StatusCancelled ||
		(from == StatusDone && to == StatusInDev) || (from == StatusCancelled && to == StatusPlanning)
}

// ensureAllowed comprueba perfil, ámbito y motivo de una transición ya válida.
func (s Stage) ensureAllowed(to Status, actor kernel.Actor, reason string) error {
	if !slices.Contains(statusProfiles[s.Status][to], actor.Profile) {
		return kernel.ErrTransitionForbidden
	}
	if actor.Profile == kernel.ProfileTL && !s.Team().IsLead(actor) {
		return kernel.ErrNotStageLead
	}
	if reasonRequired(s.Status, to) && strings.TrimSpace(reason) == "" {
		return kernel.ErrReasonRequired
	}
	return nil
}

// ErrPlanForbidden indica que el perfil no planea etapas.
var ErrPlanForbidden = kernel.Forbidden("solo PP o el usuario crean, editan, agregan, retiran o eliminan etapas")

// ErrReviseForbidden indica que el perfil no revisa etapas.
var ErrReviseForbidden = kernel.Forbidden("solo PP, el usuario o el TL de la etapa revisan una etapa")

// EnsureCanPlan exige que el actor pueda crear, editar, agregar, retirar o eliminar
// etapas. Se comprueba antes de leer o escribir en la base.
func EnsureCanPlan(actor kernel.Actor) error {
	if !actor.Profile.IsPlanner() {
		return ErrPlanForbidden
	}
	return nil
}

// EnsureCanRevise rechaza, antes de leer la etapa, a un perfil que no revisa
// etapas. Un TL además debe ser el de la etapa: ver EnsureReviser.
func EnsureCanRevise(actor kernel.Actor) error {
	if !actor.Profile.IsPlanner() && actor.Profile != kernel.ProfileTL {
		return ErrReviseForbidden
	}
	return nil
}

// EnsureReviser exige que el actor pueda revisar esta etapa: PP, el usuario o su
// TL, que la revisa para fijar el contrato.
func (s Stage) EnsureReviser(actor kernel.Actor) error {
	if err := EnsureCanRevise(actor); err != nil {
		return err
	}
	if actor.Profile == kernel.ProfileTL && !s.Team().IsLead(actor) {
		return kernel.ErrNotStageLead
	}
	return nil
}

// EnsureLeadProfile rechaza, antes de leer la etapa, a quien no es TL. Que sea el
// de la etapa se comprueba después con EnsureLead.
func EnsureLeadProfile(actor kernel.Actor) error {
	if actor.Profile != kernel.ProfileTL {
		return kernel.ErrNotStageLead
	}
	return nil
}

// EnsureCanChangeStatus rechaza, antes de leer, a un perfil que no mueve ninguna etapa,
// como PC, que solo lee.
func EnsureCanChangeStatus(actor kernel.Actor) error {
	if !kernel.HasAnyTransition(statusProfiles, actor.Profile) {
		return kernel.ErrTransitionForbidden
	}
	return nil
}
