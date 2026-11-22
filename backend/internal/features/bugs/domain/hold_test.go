package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	anaID = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	anaH  = kernel.Hold{HolderID: anaID, TakenBy: "ana"}
)

func TestChangeStatusTakesAndReleases(t *testing.T) {
	cases := []struct {
		name     string
		from, to Status
		before   kernel.Hold
		want     kernel.Hold
	}{
		{"pasarlo a in_dev lo toma", StatusConfirmed, StatusInDev, kernel.Hold{}, anaH},
		{"en revisión sigue tomado", StatusInDev, StatusInReview, anaH, anaH},
		{"con correcciones sigue tomado", StatusInReview, StatusChangesRequested, anaH, anaH},
		{"retomarlo lo toma quien lo mueve", StatusChangesRequested, StatusInDev, kernel.Hold{}, anaH},
		{"en qa sigue tomado", StatusInReview, StatusInQA, anaH, anaH},
		{"por cerrar sigue tomado", StatusInQA, StatusPendingClose, anaH, anaH},
		{"cerrarlo lo libera", StatusPendingClose, StatusDone, anaH, kernel.Hold{}},
		{"rechazarlo lo deja libre", StatusReported, StatusRejected, kernel.Hold{}, kernel.Hold{}},
		{"reabrirlo no lo toma", StatusRejected, StatusReported, kernel.Hold{}, kernel.Hold{}},
		{"confirmarlo no lo toma", StatusReported, StatusConfirmed, kernel.Hold{}, kernel.Hold{}},
	}
	for _, c := range cases {
		b := Bug{Status: c.from, Hold: c.before}
		actor := allowedActor(c.from, c.to)
		actor.UserID, actor.UserName = anaID, "ana"
		if err := b.ChangeStatus(c.to, actor, "motivo"); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if b.Hold != c.want {
			t.Errorf("%s: hold = %+v", c.name, b.Hold)
		}
	}
}

func TestReleaseLeavesBugUntaken(t *testing.T) {
	b := Bug{ID: uuid.New(), Status: StatusInQA, Hold: anaH}

	change, err := b.Release(kernel.Actor{Name: "saul", Profile: kernel.ProfilePP}, "ana se fue")

	if err != nil {
		t.Fatal(err)
	}
	if b.Taken() || change.BugID != b.ID || change.From != StatusInQA || change.To != StatusInQA || !strings.Contains(change.Reason, "ana") {
		t.Errorf("bug = %+v, registro = %+v", b, change)
	}
}

func TestReleaseRequiresReasonAndHold(t *testing.T) {
	owner := kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}
	taken := Bug{Status: StatusInDev, Hold: anaH}
	if _, err := taken.Release(owner, ""); !errors.Is(err, ErrReleaseReasonRequired) || !taken.Taken() {
		t.Errorf("sin motivo: %v", err)
	}
	free := Bug{Status: StatusInDev}
	if _, err := free.Release(owner, "motivo"); !errors.Is(err, ErrBugNotTaken) {
		t.Errorf("libre: %v", err)
	}
}

func TestEnsureCanRelease(t *testing.T) {
	if err := EnsureCanRelease(kernel.Actor{Profile: kernel.ProfilePP}); err != nil {
		t.Error(err)
	}
	if err := EnsureCanRelease(kernel.Actor{Profile: kernel.ProfileTL}); !errors.Is(err, ErrReleaseForbidden) {
		t.Errorf("TL: %v", err)
	}
}

func TestErrTakenBy(t *testing.T) {
	var err *kernel.Error
	if !errors.As(ErrTakenBy("ana"), &err) || err.Kind != kernel.KindConflict || err.Error() != "el bug lo está trabajando ana" {
		t.Errorf("err = %v", ErrTakenBy("ana"))
	}
}
