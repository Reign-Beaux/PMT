package domain

import (
	"errors"
	"testing"

	"pmt/internal/kernel"
)

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
	pp := kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	pc := kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}
	tl := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	tests := []struct {
		name   string
		from   Status
		to     Status
		actor  kernel.Actor
		reason string
		want   error
	}{
		{"PP libera", StatusPlanning, StatusStandBy, pp, "", nil},
		{"usuario libera", StatusPlanning, StatusStandBy, kernel.Owner, "", nil},
		{"PC no libera", StatusPlanning, StatusStandBy, pc, "", kernel.ErrTransitionForbidden},
		{"TL no mueve proyectos", StatusStandBy, StatusInDev, tl, "", kernel.ErrTransitionForbidden},
		{"PP detiene", StatusWorking, StatusStopped, pp, "pausa", nil},
		{"usuario detiene", StatusWorking, StatusStopped, kernel.Owner, "pausa", nil},
		{"PC no detiene", StatusWorking, StatusStopped, pc, "pausa", kernel.ErrTransitionForbidden},
		{"detener sin motivo", StatusWorking, StatusStopped, pp, "  ", kernel.ErrReasonRequired},
		{"PP reanuda", StatusStopped, StatusWorking, pp, "", nil},
		{"usuario reanuda", StatusStopped, StatusWorking, kernel.Owner, "", nil},
		{"PC no reanuda", StatusStopped, StatusWorking, pc, "", kernel.ErrTransitionForbidden},
		{"PC no cancela", StatusWorking, StatusCancelled, pc, "x", kernel.ErrTransitionForbidden},
		{"reabrir sin motivo", StatusCancelled, StatusPlanning, pp, "", kernel.ErrReasonRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Project{Status: tt.from}
			if err := p.ChangeStatus(tt.to, tt.actor, tt.reason); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
		})
	}
}

func TestEnsureCanPlanOnlyPlanners(t *testing.T) {
	for _, p := range []kernel.Profile{kernel.ProfilePP, kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDevFrontend, kernel.ProfileDevBackend, kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA, kernel.ProfileDP, kernel.ProfileUser} {
		err := EnsureCanPlan(kernel.Actor{Name: "agente", Profile: p})
		if p == kernel.ProfilePP || p == kernel.ProfileUser {
			if err != nil {
				t.Errorf("%s: err = %v", p, err)
			}
			continue
		}
		if !errors.Is(err, ErrPlanForbidden) {
			t.Errorf("%s: err = %v", p, err)
		}
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
