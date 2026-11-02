package domain

import (
	"slices"
	"strings"

	"pmt/internal/kernel"
)

var (
	devs      = []kernel.Profile{kernel.ProfileDevFrontend, kernel.ProfileDevBackend}
	reviewers = []kernel.Profile{kernel.ProfileCR}
	testers   = []kernel.Profile{kernel.ProfileQA}
	leads     = []kernel.Profile{kernel.ProfileTL}
	triagers  = []kernel.Profile{kernel.ProfileQA, kernel.ProfileTL}
	reporters = []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
)

// ErrCreateForbidden indica que el perfil no registra bugs.
var ErrCreateForbidden = kernel.Forbidden("solo PP o el usuario registran bugs")

// EnsureCanCreate exige que el actor pueda registrar un bug.
func EnsureCanCreate(actor kernel.Actor) error {
	if !slices.Contains(reporters, actor.Profile) {
		return ErrCreateForbidden
	}
	return nil
}

// ErrUpdateForbidden indica que el perfil no corrige bugs.
var ErrUpdateForbidden = kernel.Forbidden("solo PP o el usuario corrigen bugs")

// EnsureCanUpdate exige que el actor pueda corregir un bug.
func EnsureCanUpdate(actor kernel.Actor) error {
	if !slices.Contains(reporters, actor.Profile) {
		return ErrUpdateForbidden
	}
	return nil
}

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
