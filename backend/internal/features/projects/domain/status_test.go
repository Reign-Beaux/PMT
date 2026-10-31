package domain

import (
	"errors"
	"fmt"
	"testing"

	"pmt/internal/kernel"
)

func TestChangeStatusTransitions(t *testing.T) {
	allowed := map[Status]map[Status]bool{
		StatusPlanning:  {StatusStandBy: true, StatusStopped: true, StatusCancelled: true},
		StatusStandBy:   {StatusPlanning: true, StatusInDev: true, StatusStopped: true, StatusCancelled: true},
		StatusInDev:     {StatusWorking: true, StatusStopped: true, StatusCancelled: true},
		StatusWorking:   {StatusStopped: true, StatusCancelled: true},
		StatusStopped:   {StatusPlanning: true, StatusStandBy: true, StatusInDev: true, StatusWorking: true, StatusCancelled: true},
		StatusCancelled: {StatusPlanning: true},
	}
	statuses := []Status{StatusPlanning, StatusStandBy, StatusInDev, StatusWorking, StatusStopped, StatusCancelled}

	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(fmt.Sprintf("%s a %s", from, to), func(t *testing.T) {
				p := Project{Status: from}

				err := p.ChangeStatus(to, kernel.Owner, "motivo")

				if allowed[from][to] {
					if err != nil || p.Status != to {
						t.Fatalf("err = %v, status = %s", err, p.Status)
					}
					return
				}
				if !errors.Is(err, ErrProjectStatusTransition) || p.Status != from {
					t.Fatalf("err = %v, status = %s", err, p.Status)
				}
			})
		}
	}
}

func TestChangeStatusRejectsUnknownStatus(t *testing.T) {
	for _, status := range []Status{"", "archived", "PLANNING", " planning "} {
		p := Project{Status: StatusPlanning}

		if err := p.ChangeStatus(status, kernel.Owner, "motivo"); !errors.Is(err, ErrProjectStatusInvalid) {
			t.Errorf("status %q: err = %v", status, err)
		}
		if p.Status != StatusPlanning {
			t.Errorf("status %q: proyecto = %+v", status, p)
		}
	}
}
