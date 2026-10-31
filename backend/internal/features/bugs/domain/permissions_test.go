package domain

import (
	"errors"
	"testing"

	"pmt/internal/kernel"
)

func allowedActor(from, to Status) kernel.Actor {
	if profiles := statusProfiles[from][to]; len(profiles) > 0 {
		return kernel.Actor{Name: "agente", Profile: profiles[0]}
	}
	return kernel.Owner
}

func TestEveryTransitionHasProfiles(t *testing.T) {
	for from, targets := range statusTransitions {
		for _, to := range targets {
			if len(statusProfiles[from][to]) == 0 {
				t.Errorf("%s a %s no tiene perfiles", from, to)
			}
		}
		if len(statusProfiles[from]) != len(targets) {
			t.Errorf("%s tiene perfiles para transiciones que no existen", from)
		}
	}
}

func TestChangeStatusValidatesProfile(t *testing.T) {
	actor := func(p kernel.Profile) kernel.Actor { return kernel.Actor{Name: "a", Profile: p} }
	tests := []struct {
		name   string
		from   Status
		to     Status
		actor  kernel.Actor
		reason string
		want   error
	}{
		{"QA confirma", StatusReported, StatusConfirmed, actor(kernel.ProfileQA), "", nil},
		{"dev no confirma", StatusReported, StatusConfirmed, actor(kernel.ProfileDevBackend), "", kernel.ErrTransitionForbidden},
		{"cualquier dev inicia", StatusConfirmed, StatusInProgress, actor(kernel.ProfileDevFrontend), "", nil},
		{"CR aprueba", StatusInReview, StatusInQA, actor(kernel.ProfileCR), "", nil},
		{"QA cierra", StatusInQA, StatusDone, actor(kernel.ProfileQA), "", nil},
		{"usuario no cierra", StatusInQA, StatusDone, kernel.Owner, "", kernel.ErrTransitionForbidden},
		{"QA rechaza reportado", StatusReported, StatusRejected, actor(kernel.ProfileQA), "duplicado", nil},
		{"QA no rechaza en curso", StatusInProgress, StatusRejected, actor(kernel.ProfileQA), "x", kernel.ErrTransitionForbidden},
		{"TL rechaza en curso", StatusInProgress, StatusRejected, actor(kernel.ProfileTL), "x", nil},
		{"rechazar sin motivo", StatusReported, StatusRejected, actor(kernel.ProfileTL), " ", kernel.ErrReasonRequired},
		{"reabrir sin motivo", StatusRejected, StatusReported, actor(kernel.ProfileQA), "", kernel.ErrReasonRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bug := Bug{Status: tt.from}
			if err := bug.ChangeStatus(tt.to, tt.actor, tt.reason); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
		})
	}
}
