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
		{"cualquier dev inicia", StatusConfirmed, StatusInDev, actor(kernel.ProfileDevFrontend), "", nil},
		{"CR aprueba", StatusInReview, StatusInQA, actor(kernel.ProfileCR), "", nil},
		{"QA aprueba", StatusInQA, StatusPendingClose, actor(kernel.ProfileQA), "verificado en /tmp/qa.log", nil},
		{"aprobar sin motivo", StatusInQA, StatusPendingClose, actor(kernel.ProfileQA), "", kernel.ErrReasonRequired},
		{"aprobar con solo espacios", StatusInQA, StatusPendingClose, actor(kernel.ProfileQA), "  ", kernel.ErrReasonRequired},
		{"CR no aprueba por QA", StatusInQA, StatusPendingClose, actor(kernel.ProfileCR), "verificado", kernel.ErrTransitionForbidden},
		{"CR cierra cambio interno", StatusInReview, StatusPendingClose, actor(kernel.ProfileCR), "interno", nil},
		{"CR sin motivo", StatusInReview, StatusPendingClose, actor(kernel.ProfileCR), " ", kernel.ErrReasonRequired},
		{"TL cierra", StatusPendingClose, StatusDone, actor(kernel.ProfileTL), "", nil},
		{"QA no cierra", StatusPendingClose, StatusDone, actor(kernel.ProfileQA), "", kernel.ErrTransitionForbidden},
		{"usuario no cierra", StatusPendingClose, StatusDone, kernel.Owner, "", kernel.ErrTransitionForbidden},
		{"TL devuelve a CR", StatusPendingClose, StatusInReview, actor(kernel.ProfileTL), "falta", nil},
		{"TL devuelve a QA", StatusPendingClose, StatusInQA, actor(kernel.ProfileTL), "falta", nil},
		{"devolver sin motivo", StatusPendingClose, StatusInQA, actor(kernel.ProfileTL), "", kernel.ErrReasonRequired},
		{"TL rechaza en cierre", StatusPendingClose, StatusRejected, actor(kernel.ProfileTL), "x", nil},
		{"dev reentrega por in_dev", StatusChangesRequested, StatusInDev, actor(kernel.ProfileDevBackend), "", nil},
		{"QA rechaza reportado", StatusReported, StatusRejected, actor(kernel.ProfileQA), "duplicado", nil},
		{"QA no rechaza en curso", StatusInDev, StatusRejected, actor(kernel.ProfileQA), "x", kernel.ErrTransitionForbidden},
		{"TL rechaza en curso", StatusInDev, StatusRejected, actor(kernel.ProfileTL), "x", nil},
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

func TestEnsureCanChangeStatusRejectsPC(t *testing.T) {
	if err := EnsureCanChangeStatus(kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}); !errors.Is(err, kernel.ErrTransitionForbidden) {
		t.Errorf("PC: err = %v", err)
	}
	for _, targets := range statusProfiles {
		for _, profiles := range targets {
			for _, p := range profiles {
				if err := EnsureCanChangeStatus(kernel.Actor{Name: "agente", Profile: p}); err != nil {
					t.Errorf("%s: err = %v", p, err)
				}
			}
		}
	}
}
