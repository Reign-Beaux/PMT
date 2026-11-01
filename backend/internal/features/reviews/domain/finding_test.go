package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	cr = kernel.Actor{Name: "cr", Profile: kernel.ProfileCR}
	qa = kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
	tl = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	pp = kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	us = kernel.Owner
)

func TestEnsureCanCreate(t *testing.T) {
	for _, actor := range []kernel.Actor{cr, qa} {
		if err := EnsureCanCreate(actor); err != nil {
			t.Errorf("%s: %v", actor.Name, err)
		}
	}
	for _, actor := range []kernel.Actor{tl, pp, us} {
		if err := EnsureCanCreate(actor); !errors.Is(err, ErrCreateForbidden) {
			t.Errorf("%s: %v", actor.Name, err)
		}
	}
}

func TestEnsureKind(t *testing.T) {
	if err := EnsureKind(ItemBug); err != nil {
		t.Error(err)
	}
	if err := EnsureKind("stage"); !errors.Is(err, ErrItemKindInvalid) {
		t.Errorf("err = %v", err)
	}
}

func TestNewFinding(t *testing.T) {
	f, err := NewFinding(ItemTask, uuid.New(), "  Falta validar  ", true, cr)
	if err != nil {
		t.Fatal(err)
	}
	if f.Description != "Falta validar" || f.Status != StatusOpen || !f.Blocking || f.ActorName != "cr" || f.ActorProfile != kernel.ProfileCR {
		t.Errorf("hallazgo = %+v", f)
	}
	if _, err := NewFinding(ItemTask, uuid.New(), "   ", true, cr); !errors.Is(err, ErrDescriptionRequired) {
		t.Errorf("err = %v", err)
	}
}

func TestClose(t *testing.T) {
	tests := []struct {
		name       string
		status     Status
		resolution Status
		actor      kernel.Actor
		reason     string
		want       error
	}{
		{"CR corrige", StatusOpen, StatusFixed, cr, "Listo", nil},
		{"QA corrige", StatusOpen, StatusFixed, qa, "Listo", nil},
		{"TL descarta", StatusOpen, StatusDismissed, tl, "No aplica", nil},
		{"TL no corrige", StatusOpen, StatusFixed, tl, "Listo", ErrFixForbidden},
		{"CR no descarta", StatusOpen, StatusDismissed, cr, "No aplica", ErrDismissForbidden},
		{"User no descarta", StatusOpen, StatusDismissed, us, "No aplica", ErrDismissForbidden},
		{"resolución desconocida", StatusOpen, "open", cr, "Listo", ErrResolutionInvalid},
		{"sin motivo", StatusOpen, StatusFixed, cr, "   ", ErrCloseReasonRequired},
		{"ya corregido", StatusFixed, StatusDismissed, tl, "No aplica", ErrFindingClosed},
		{"ya descartado", StatusDismissed, StatusFixed, cr, "Listo", ErrFindingClosed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Finding{Status: tt.status}

			err := f.Close(tt.resolution, tt.actor, tt.reason)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if tt.want == nil && (f.Status != tt.resolution || f.ClosedBy != tt.actor.Name || f.CloseReason != tt.reason) {
				t.Errorf("hallazgo = %+v", f)
			}
		})
	}
}

func TestNeedsStageLead(t *testing.T) {
	task, bug := Finding{ItemKind: ItemTask}, Finding{ItemKind: ItemBug}
	if !task.NeedsStageLead(StatusDismissed) || task.NeedsStageLead(StatusFixed) || bug.NeedsStageLead(StatusDismissed) {
		t.Error("solo descartar en una tarea exige ser el TL de la etapa")
	}
}
