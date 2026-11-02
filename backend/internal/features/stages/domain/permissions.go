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
	stoppers = []kernel.Profile{kernel.ProfilePP, kernel.ProfileTL, kernel.ProfilePC, kernel.ProfileUser}
)

// statusProfiles indica qué perfiles pueden hacer cada transición. User tiene lo de
// PP y PC. Un TL además debe ser el de la etapa.
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
