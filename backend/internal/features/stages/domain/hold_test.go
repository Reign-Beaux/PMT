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
	beaID = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	anaH  = kernel.Hold{HolderID: anaID, TakenBy: "ana"}
)

// persona es el actor que permite la transición, con el usuario de la persona.
func persona(s Stage, to Status, id uuid.UUID, name string) kernel.Actor {
	actor := allowedActor(s, to)
	actor.UserID, actor.UserName = id, name
	return actor
}

func TestChangeStatusTakesAndReleases(t *testing.T) {
	cases := []struct {
		name     string
		from, to Status
		before   kernel.Hold
		want     kernel.Hold
	}{
		{"iniciar la toma quien la inicia", StatusStandBy, StatusInDev, kernel.Hold{}, anaH},
		{"detenida sigue tomada", StatusInDev, StatusStopped, anaH, anaH},
		{"reanudarla la toma quien la mueve", StatusStopped, StatusInDev, kernel.Hold{}, anaH},
		{"reabrirla desde done la toma quien la mueve", StatusDone, StatusInDev, kernel.Hold{}, anaH},
		{"cerrarla la libera", StatusInDev, StatusDone, anaH, kernel.Hold{}},
		{"cancelarla la libera", StatusInDev, StatusCancelled, anaH, kernel.Hold{}},
		{"cancelarla detenida la libera", StatusStopped, StatusCancelled, anaH, kernel.Hold{}},
		{"volver a planning la libera", StatusStopped, StatusPlanning, anaH, kernel.Hold{}},
		{"volver a stand_by la libera", StatusStopped, StatusStandBy, anaH, kernel.Hold{}},
		{"reabrir una cancelada a planning no la toma", StatusCancelled, StatusPlanning, kernel.Hold{}, kernel.Hold{}},
	}
	for _, c := range cases {
		s := Stage{Status: c.from, Lead: "tl", Hold: c.before}
		if err := s.ChangeStatus(c.to, persona(s, c.to, anaID, "ana"), "motivo"); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if s.Hold != c.want {
			t.Errorf("%s: hold = %+v", c.name, s.Hold)
		}
	}
}

func TestChangeStatusWithoutUserTakesNothing(t *testing.T) {
	s := Stage{Status: StatusStandBy, Lead: "tl"}
	if err := s.ChangeStatus(StatusInDev, allowedActor(s, StatusInDev), ""); err != nil || s.Taken() {
		t.Fatalf("hold = %+v, err = %v", s.Hold, err)
	}
}

func TestReleaseLeavesStageUntaken(t *testing.T) {
	s := Stage{ID: uuid.New(), Status: StatusStopped, Hold: anaH}
	owner := kernel.Actor{Name: "saul", Profile: kernel.ProfileUser, UserID: beaID}

	change, err := s.Release(owner, "  ana se fue  ")

	if err != nil {
		t.Fatal(err)
	}
	if s.Taken() {
		t.Errorf("sigue tomada: %+v", s.Hold)
	}
	if change.StageID != s.ID || change.From != StatusStopped || change.To != StatusStopped || change.ActorName != "saul" {
		t.Errorf("registro = %+v", change)
	}
	if !strings.Contains(change.Reason, "ana") || !strings.HasSuffix(change.Reason, ": ana se fue") {
		t.Errorf("motivo = %q", change.Reason)
	}
}

func TestReleaseRequiresReasonAndHold(t *testing.T) {
	owner := kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}
	taken := Stage{Status: StatusInDev, Hold: anaH}
	if _, err := taken.Release(owner, " "); !errors.Is(err, ErrReleaseReasonRequired) || !taken.Taken() {
		t.Errorf("sin motivo: %v", err)
	}
	free := Stage{Status: StatusInDev}
	if _, err := free.Release(owner, "motivo"); !errors.Is(err, ErrStageNotTaken) {
		t.Errorf("libre: %v", err)
	}
}

func TestEnsureCanRelease(t *testing.T) {
	for _, p := range []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser} {
		if err := EnsureCanRelease(kernel.Actor{Profile: p}); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	for _, p := range []kernel.Profile{kernel.ProfileTL, kernel.ProfilePC, kernel.ProfileDevBackend, kernel.ProfileQA, ""} {
		if err := EnsureCanRelease(kernel.Actor{Profile: p}); !errors.Is(err, ErrReleaseForbidden) {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestErrTakenBy(t *testing.T) {
	var err *kernel.Error
	if !errors.As(ErrTakenBy("ana"), &err) || err.Kind != kernel.KindConflict || err.Error() != "la etapa la está trabajando ana" {
		t.Errorf("err = %v", err)
	}
	if ErrTakenBy("").Error() != "la etapa la está trabajando otra persona" {
		t.Errorf("sin nombre: %v", ErrTakenBy(""))
	}
}
