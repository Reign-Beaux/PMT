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
		StatusReady:            {StatusInDev: true, StatusCancelled: true},
		StatusInDev:            {StatusInReview: true, StatusCancelled: true},
		StatusInReview:         {StatusInQA: true, StatusChangesRequested: true, StatusPendingClose: true, StatusCancelled: true},
		StatusChangesRequested: {StatusInDev: true, StatusCancelled: true},
		StatusInQA:             {StatusPendingClose: true, StatusChangesRequested: true, StatusCancelled: true},
		StatusPendingClose:     {StatusDone: true, StatusInReview: true, StatusInQA: true, StatusCancelled: true},
		StatusDone:             {},
		StatusCancelled:        {StatusBacklog: true},
	}
	statuses := []Status{
		StatusBacklog, StatusReady, StatusInDev, StatusInReview, StatusChangesRequested,
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

func TestReviseStatus(t *testing.T) {
	tests := []struct {
		from Status
		want Status
		err  error
	}{
		{StatusInReview, StatusChangesRequested, nil},
		{StatusInQA, StatusChangesRequested, nil},
		{StatusPendingClose, StatusChangesRequested, nil},
		{StatusDone, "", ErrTaskDoneRevised},
		{StatusBacklog, StatusBacklog, nil},
		{StatusInDev, StatusInDev, nil},
		{StatusChangesRequested, StatusChangesRequested, nil},
		{StatusCancelled, StatusCancelled, nil},
	}
	for _, tt := range tests {
		t.Run(string(tt.from), func(t *testing.T) {
			got, err := Task{Status: tt.from}.ReviseStatus()
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Fatalf("got = %q, err = %v", got, err)
			}
		})
	}
}

func TestClosureEnsureClosable(t *testing.T) {
	tests := []struct {
		name    string
		closure Closure
		want    error
	}{
		{"terminada", Closure{Done: 2}, nil},
		{"pendientes", Closure{Done: 1, Pending: 1}, ErrStageTasksPending},
		{"vacía o toda cancelada", Closure{}, ErrStageNoTasksDone},
		{"sin done y con pendientes", Closure{Pending: 1}, ErrStageTasksPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.closure.EnsureClosable(); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
