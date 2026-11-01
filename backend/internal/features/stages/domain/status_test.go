package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func TestNewStageStartsInPlanning(t *testing.T) {
	s, err := NewStage(uuid.New(), "Diseño", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusPlanning {
		t.Errorf("status = %s", s.Status)
	}
}

func TestChangeStatusTransitions(t *testing.T) {
	allowed := map[Status]map[Status]bool{
		StatusPlanning:  {StatusStandBy: true, StatusStopped: true, StatusCancelled: true},
		StatusStandBy:   {StatusPlanning: true, StatusInDev: true, StatusStopped: true, StatusCancelled: true},
		StatusInDev:     {StatusDone: true, StatusStopped: true, StatusCancelled: true},
		StatusDone:      {StatusInDev: true},
		StatusStopped:   {StatusPlanning: true, StatusStandBy: true, StatusInDev: true, StatusCancelled: true},
		StatusCancelled: {StatusPlanning: true},
	}
	statuses := []Status{StatusPlanning, StatusStandBy, StatusInDev, StatusDone, StatusStopped, StatusCancelled}

	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(fmt.Sprintf("%s a %s", from, to), func(t *testing.T) {
				s := Stage{Status: from, Lead: "tl"}

				err := s.ChangeStatus(to, allowedActor(s, to), "motivo")

				if allowed[from][to] {
					if err != nil || s.Status != to {
						t.Fatalf("err = %v, status = %s", err, s.Status)
					}
					return
				}
				if !errors.Is(err, ErrStageStatusTransition) || s.Status != from {
					t.Fatalf("err = %v, status = %s", err, s.Status)
				}
			})
		}
	}
}

func TestChangeStatusRejectsUnknownStatus(t *testing.T) {
	for _, status := range []Status{"", "archived", "PLANNING", " planning "} {
		s := Stage{Status: StatusPlanning}

		if err := s.ChangeStatus(status, allowedActor(s, status), "motivo"); !errors.Is(err, ErrStageStatusInvalid) {
			t.Errorf("status %q: err = %v", status, err)
		}
		if s.Status != StatusPlanning {
			t.Errorf("status %q: etapa = %+v", status, s)
		}
	}
}

func TestEndsVersionOne(t *testing.T) {
	closing := Stage{ID: uuid.New(), Status: StatusCancelled}
	done := Stage{ID: uuid.New(), Status: StatusDone}
	cancelled := Stage{ID: uuid.New(), Status: StatusCancelled}
	open := Stage{ID: uuid.New(), Status: StatusInDev}
	stopped := Stage{ID: uuid.New(), Status: StatusStopped}
	added := Stage{ID: uuid.New(), Status: StatusCancelled}
	closed := Stage{ID: uuid.New(), Status: StatusDone}

	tests := []struct {
		name       string
		stage      Stage
		versionOne []Stage
		want       bool
	}{
		{"cancelar la última con otra en done", closing, []Stage{done, closing}, true},
		{"cancelar con otra abierta", closing, []Stage{done, open, closing}, false},
		{"cancelar con otra detenida", closing, []Stage{done, stopped, closing}, false},
		{"todas canceladas", closing, []Stage{cancelled, closing}, false},
		{"cancelar una agregada no termina la versión 1", added, []Stage{done, closing}, false},
		{"cerrar la última", closed, []Stage{cancelled, closed}, true},
		{"cerrar una agregada con la versión 1 terminada", closed, []Stage{done}, true},
		{"cerrar con otra abierta", closed, []Stage{closed, open}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EndsVersionOne(tt.stage, tt.versionOne); got != tt.want {
				t.Errorf("EndsVersionOne = %v, se esperaba %v", got, tt.want)
			}
		})
	}
}

func TestRetirementEndsVersionOne(t *testing.T) {
	retired := Stage{ID: uuid.New(), Status: StatusInDev}
	done := Stage{ID: uuid.New(), Status: StatusDone}
	cancelled := Stage{ID: uuid.New(), Status: StatusCancelled}
	open := Stage{ID: uuid.New(), Status: StatusInDev}
	stopped := Stage{ID: uuid.New(), Status: StatusStopped}

	tests := []struct {
		name       string
		stageID    uuid.UUID
		versionOne []Stage
		want       bool
	}{
		{"retirar la última con otra en done", retired.ID, []Stage{done, retired}, true},
		{"retirar la última con done y cancelled", retired.ID, []Stage{done, cancelled, retired}, true},
		{"retirar con otra abierta", retired.ID, []Stage{done, open, retired}, false},
		{"retirar con otra detenida", retired.ID, []Stage{done, stopped, retired}, false},
		{"sin ninguna en done", retired.ID, []Stage{cancelled, retired}, false},
		{"retirar la única", retired.ID, []Stage{retired}, false},
		{"retirar la única en done", done.ID, []Stage{done, cancelled}, false},
		{"retirar una agregada no termina la versión 1", uuid.New(), []Stage{done}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RetirementEndsVersionOne(tt.stageID, tt.versionOne); got != tt.want {
				t.Errorf("RetirementEndsVersionOne = %v, se esperaba %v", got, tt.want)
			}
		})
	}
}
