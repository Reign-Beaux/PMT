package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestStatusValues(t *testing.T) {
	want := map[Status]string{
		StatusReported:         "reported",
		StatusConfirmed:        "confirmed",
		StatusInProgress:       "in_progress",
		StatusInReview:         "in_review",
		StatusChangesRequested: "changes_requested",
		StatusInQA:             "in_qa",
		StatusDone:             "done",
		StatusRejected:         "rejected",
	}
	if len(want) != 8 {
		t.Fatalf("status repetidos: %d", len(want))
	}
	for status, value := range want {
		if string(status) != value {
			t.Errorf("%q != %q", status, value)
		}
	}
}

func TestChangeStatusTransitions(t *testing.T) {
	allowed := map[Status]map[Status]bool{
		StatusReported:         {StatusConfirmed: true, StatusRejected: true},
		StatusConfirmed:        {StatusInProgress: true, StatusRejected: true},
		StatusInProgress:       {StatusInReview: true, StatusRejected: true},
		StatusInReview:         {StatusInQA: true, StatusChangesRequested: true, StatusRejected: true},
		StatusChangesRequested: {StatusInReview: true, StatusRejected: true},
		StatusInQA:             {StatusDone: true, StatusChangesRequested: true, StatusRejected: true},
		StatusDone:             {},
		StatusRejected:         {StatusReported: true},
	}
	statuses := []Status{
		StatusReported, StatusConfirmed, StatusInProgress, StatusInReview,
		StatusChangesRequested, StatusInQA, StatusDone, StatusRejected,
	}

	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(fmt.Sprintf("%s a %s", from, to), func(t *testing.T) {
				bug := Bug{Status: from}

				err := bug.ChangeStatus(to)

				if allowed[from][to] {
					if err != nil || bug.Status != to {
						t.Fatalf("err = %v, status = %s", err, bug.Status)
					}
					return
				}
				if !errors.Is(err, ErrBugStatusTransition) || bug.Status != from {
					t.Fatalf("err = %v, status = %s", err, bug.Status)
				}
			})
		}
	}
}

func TestChangeStatusRejectsUnknownStatus(t *testing.T) {
	for _, status := range []Status{"", "archived", "REPORTED", " confirmed "} {
		bug := Bug{Status: StatusReported}

		if err := bug.ChangeStatus(status); !errors.Is(err, ErrBugStatusInvalid) {
			t.Errorf("status %q: err = %v", status, err)
		}
		if bug.Status != StatusReported {
			t.Errorf("status %q: bug = %+v", status, bug)
		}
	}
}
