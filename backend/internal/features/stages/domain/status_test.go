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
