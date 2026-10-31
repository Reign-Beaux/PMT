package domain

import (
	"errors"
	"fmt"
	"testing"

	"pmt/internal/kernel"
)

func TestChangeStatusTransitions(t *testing.T) {
	allowed := map[Status]map[Status]bool{
		StatusBacklog:          {StatusReady: true, StatusCancelled: true},
		StatusReady:            {StatusInProgress: true, StatusCancelled: true},
		StatusInProgress:       {StatusInReview: true, StatusCancelled: true},
		StatusInReview:         {StatusInQA: true, StatusChangesRequested: true, StatusCancelled: true},
		StatusChangesRequested: {StatusInReview: true, StatusCancelled: true},
		StatusInQA:             {StatusPendingClose: true, StatusChangesRequested: true, StatusCancelled: true},
		StatusPendingClose:     {StatusDone: true, StatusCancelled: true},
		StatusDone:             {},
		StatusCancelled:        {StatusBacklog: true},
	}
	statuses := []Status{
		StatusBacklog, StatusReady, StatusInProgress, StatusInReview, StatusChangesRequested,
		StatusInQA, StatusPendingClose, StatusDone, StatusCancelled,
	}

	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(fmt.Sprintf("%s a %s", from, to), func(t *testing.T) {
				task, actor, team := allowedChange(from, to)

				err := task.ChangeStatus(to, actor, "motivo", team)

				if allowed[from][to] {
					if err != nil || task.Status != to {
						t.Fatalf("err = %v, status = %s", err, task.Status)
					}
					return
				}
				if !errors.Is(err, ErrTaskStatusTransition) || task.Status != from {
					t.Fatalf("err = %v, status = %s", err, task.Status)
				}
			})
		}
	}
}

func TestChangeStatusRejectsUnknownStatus(t *testing.T) {
	for _, status := range []Status{"", "archived", "BACKLOG", " ready "} {
		task := Task{Status: StatusBacklog}

		if err := task.ChangeStatus(status, kernel.Owner, "motivo", kernel.Team{}); !errors.Is(err, ErrTaskStatusInvalid) {
			t.Errorf("status %q: err = %v", status, err)
		}
		if task.Status != StatusBacklog {
			t.Errorf("status %q: tarea = %+v", status, task)
		}
	}
}
