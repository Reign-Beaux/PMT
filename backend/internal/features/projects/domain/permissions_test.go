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
		{"PC detiene", StatusWorking, StatusStopped, pc, "pausa", nil},
		{"detener sin motivo", StatusWorking, StatusStopped, pc, "  ", kernel.ErrReasonRequired},
		{"PC reanuda", StatusStopped, StatusWorking, pc, "", nil},
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
