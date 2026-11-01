package domain

import (
	"strings"

	"pmt/internal/kernel"
)

var (
	planners = []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
	stoppers = []kernel.Profile{kernel.ProfilePP, kernel.ProfilePC, kernel.ProfileUser}
)

// statusProfiles indica qué perfiles pueden hacer cada transición. User tiene lo de PP y PC.
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
