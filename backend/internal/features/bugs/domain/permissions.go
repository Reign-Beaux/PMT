package domain

import (
	"strings"

	"pmt/internal/kernel"
)

var (
	devs      = []kernel.Profile{kernel.ProfileDevFrontend, kernel.ProfileDevBackend}
	reviewers = []kernel.Profile{kernel.ProfileCR}
	testers   = []kernel.Profile{kernel.ProfileQA}
	leads     = []kernel.Profile{kernel.ProfileTL}
	triagers  = []kernel.Profile{kernel.ProfileQA, kernel.ProfileTL}
)

// statusProfiles indica qué perfiles pueden hacer cada transición. Un bug no tiene
// etapa: todos son por perfil, sin asignación. Dev es DevFrontend o DevBackend.
var statusProfiles = map[Status]map[Status][]kernel.Profile{
	StatusReported:         {StatusConfirmed: testers, StatusRejected: triagers},
	StatusConfirmed:        {StatusInDev: devs, StatusRejected: leads},
	StatusInDev:            {StatusInReview: devs, StatusRejected: leads},
	StatusInReview:         {StatusInQA: reviewers, StatusChangesRequested: reviewers, StatusPendingClose: reviewers, StatusRejected: leads},
	StatusChangesRequested: {StatusInDev: devs, StatusRejected: leads},
	StatusInQA:             {StatusPendingClose: testers, StatusChangesRequested: testers, StatusRejected: leads},
	StatusPendingClose:     {StatusDone: leads, StatusInReview: leads, StatusInQA: leads, StatusRejected: leads},
	StatusDone:             {},
	StatusRejected:         {StatusReported: triagers},
}

func ensureReason(from, to Status, reason string) error {
	required := to == StatusRejected || (from == StatusRejected && to == StatusReported) ||
		to == StatusPendingClose ||
		(from == StatusPendingClose && (to == StatusInReview || to == StatusInQA))
	if required && strings.TrimSpace(reason) == "" {
		return kernel.ErrReasonRequired
	}
	return nil
}
