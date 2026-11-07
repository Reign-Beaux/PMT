package domain

import (
	"errors"
	"testing"

	"pmt/internal/kernel"
)

// allowedActor devuelve un actor con perfil para la transición. Si es TL, los
// casos usan una etapa cuyo TL se llama igual.
func allowedActor(s Stage, to Status) kernel.Actor {
	if profiles := statusProfiles[s.Status][to]; len(profiles) > 0 && profiles[0] != kernel.ProfileTL {
		return kernel.Actor{Name: "agente", Profile: profiles[0]}
	}
	return kernel.Actor{Name: s.Lead, Profile: kernel.ProfileTL}
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
	tl := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	otherTL := kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}
	pp := kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	pc := kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}

	tests := []struct {
		name   string
		from   Status
		to     Status
		actor  kernel.Actor
		reason string
		want   error
	}{
		{"PP libera", StatusPlanning, StatusStandBy, pp, "", nil},
		{"TL no libera", StatusPlanning, StatusStandBy, tl, "", kernel.ErrTransitionForbidden},
		{"TL toma", StatusStandBy, StatusInDev, tl, "", nil},
		{"TL ajeno no toma", StatusStandBy, StatusInDev, otherTL, "", kernel.ErrNotStageLead},
		{"PP no toma", StatusStandBy, StatusInDev, pp, "", kernel.ErrTransitionForbidden},
		{"usuario no toma", StatusStandBy, StatusInDev, kernel.Owner, "", kernel.ErrTransitionForbidden},
		{"TL cierra", StatusInDev, StatusDone, tl, "", nil},
		{"usuario no cierra", StatusInDev, StatusDone, kernel.Owner, "", kernel.ErrTransitionForbidden},
		{"reabrir sin motivo", StatusDone, StatusInDev, pp, "", kernel.ErrReasonRequired},
		{"TL detiene", StatusInDev, StatusStopped, tl, "pausa", nil},
		{"usuario detiene", StatusInDev, StatusStopped, kernel.Owner, "pausa", nil},
		{"PC no detiene", StatusInDev, StatusStopped, pc, "pausa", kernel.ErrTransitionForbidden},
		{"detener sin motivo", StatusInDev, StatusStopped, pp, " ", kernel.ErrReasonRequired},
		{"PP reanuda", StatusStopped, StatusInDev, pp, "", nil},
		{"usuario reanuda", StatusStopped, StatusInDev, kernel.Owner, "", nil},
		{"PC no reanuda", StatusStopped, StatusInDev, pc, "", kernel.ErrTransitionForbidden},
		{"PC no cancela", StatusInDev, StatusCancelled, pc, "x", kernel.ErrTransitionForbidden},
		{"usuario cancela", StatusInDev, StatusCancelled, kernel.Owner, "x", nil},
		{"TL ajeno no detiene", StatusInDev, StatusStopped, otherTL, "x", kernel.ErrNotStageLead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stage{Status: tt.from, Lead: "tl"}
			if err := s.ChangeStatus(tt.to, tt.actor, tt.reason); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
		})
	}
}

func TestStageWithoutLeadRejectsEveryTL(t *testing.T) {
	s := Stage{Status: StatusStandBy}
	err := s.ChangeStatus(StatusInDev, kernel.Actor{Profile: kernel.ProfileTL}, "")
	if !errors.Is(err, kernel.ErrNotStageLead) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureCanAssignLead(t *testing.T) {
	for _, actor := range []kernel.Actor{{Profile: kernel.ProfilePP}, kernel.Owner} {
		if err := EnsureCanAssignLead(actor); err != nil {
			t.Fatalf("%v: %v", actor, err)
		}
	}
	for _, profile := range []kernel.Profile{kernel.ProfileTL, kernel.ProfilePC, kernel.ProfileDevBackend} {
		if err := EnsureCanAssignLead(kernel.Actor{Profile: profile}); !errors.Is(err, ErrAssignLeadForbidden) {
			t.Fatalf("%s: %v", profile, err)
		}
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

func TestEnsureReviser(t *testing.T) {
	s := Stage{Lead: "tl"}
	tests := []struct {
		name  string
		actor kernel.Actor
		want  error
	}{
		{"PP", kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}, nil},
		{"usuario", kernel.Owner, nil},
		{"TL de la etapa", kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}, nil},
		{"TL de otra etapa", kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}, kernel.ErrNotStageLead},
		{"PC", kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}, ErrReviseForbidden},
		{"Dev", kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}, ErrReviseForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.EnsureReviser(tt.actor); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, quiero %v", err, tt.want)
			}
		})
	}
}

func TestEnsureLeadProfile(t *testing.T) {
	if err := EnsureLeadProfile(kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}); err != nil {
		t.Errorf("TL: err = %v", err)
	}
	if err := EnsureLeadProfile(kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}); !errors.Is(err, kernel.ErrNotStageLead) {
		t.Errorf("PC: err = %v", err)
	}
}
