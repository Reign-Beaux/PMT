package domain

import "testing"

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
