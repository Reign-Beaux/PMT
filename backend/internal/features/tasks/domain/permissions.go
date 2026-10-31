package domain

import (
	"slices"
	"strings"

	"pmt/internal/kernel"
)

var (
	// ErrNotStageMember indica que el dev no pertenece al equipo de la etapa.
	ErrNotStageMember = kernel.Forbidden("no estás en el equipo de la etapa")
	// ErrNotAssignee indica que la tarea no está a nombre de quien la mueve.
	ErrNotAssignee = kernel.Forbidden("la tarea es de otro dev")
)

var (
	leads     = []kernel.Profile{kernel.ProfileTL}
	devs      = []kernel.Profile{kernel.ProfileDevFrontend, kernel.ProfileDevBackend}
	reviewers = []kernel.Profile{kernel.ProfileCR}
	testers   = []kernel.Profile{kernel.ProfileQA}
	scope     = []kernel.Profile{kernel.ProfileTL, kernel.ProfilePP, kernel.ProfileUser}
)

// statusProfiles indica qué perfiles pueden hacer cada transición. Dev es
// DevFrontend o DevBackend. Un TL además debe ser el de la etapa, y un dev, de su
// equipo para tomarla y su dueño para moverla.
var statusProfiles = map[Status]map[Status][]kernel.Profile{
	StatusBacklog:          {StatusReady: leads, StatusCancelled: scope},
	StatusReady:            {StatusInProgress: devs, StatusCancelled: scope},
	StatusInProgress:       {StatusInReview: devs, StatusCancelled: scope},
	StatusInReview:         {StatusInQA: reviewers, StatusChangesRequested: reviewers, StatusCancelled: scope},
	StatusChangesRequested: {StatusInReview: devs, StatusCancelled: scope},
	StatusInQA:             {StatusPendingClose: testers, StatusChangesRequested: testers, StatusCancelled: scope},
	StatusPendingClose:     {StatusDone: leads, StatusCancelled: scope},
	StatusDone:             {},
	StatusCancelled:        {StatusBacklog: scope},
}

func reasonRequired(from, to Status) bool {
	return to == StatusCancelled || (from == StatusCancelled && to == StatusBacklog)
}

// EnsureProfile comprueba, sin consultar el equipo, que el status exista, que la
// transición esté permitida y que el perfil pueda hacerla.
func (t Task) EnsureProfile(status Status, actor kernel.Actor) error {
	if _, known := statusTransitions[status]; !known {
		return ErrTaskStatusInvalid
	}
	if !slices.Contains(statusTransitions[t.Status], status) {
		return ErrTaskStatusTransition
	}
	if !slices.Contains(statusProfiles[t.Status][status], actor.Profile) {
		return kernel.ErrTransitionForbidden
	}
	return nil
}

// ensureScope comprueba que quien actúa tenga la tarea a su alcance.
func (t Task) ensureScope(actor kernel.Actor, team kernel.Team) error {
	switch {
	case actor.Profile == kernel.ProfileTL:
		if !team.IsLead(actor) {
			return kernel.ErrNotStageLead
		}
	case actor.Profile.IsDev() && t.Status == StatusReady:
		if !team.HasMember(actor.Name) {
			return ErrNotStageMember
		}
	case actor.Profile.IsDev():
		if t.Assignee != actor.Name {
			return ErrNotAssignee
		}
	}
	return nil
}

func ensureReason(from, to Status, reason string) error {
	if reasonRequired(from, to) && strings.TrimSpace(reason) == "" {
		return kernel.ErrReasonRequired
	}
	return nil
}
