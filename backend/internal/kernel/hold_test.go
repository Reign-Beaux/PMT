package kernel

import (
	"testing"

	"github.com/google/uuid"
)

func TestHoldOf(t *testing.T) {
	id := uuid.New()
	hold := HoldOf(Actor{Name: "dbp-tl", UserID: id, UserName: "saul"})
	if hold.HolderID != id || hold.TakenBy != "saul" || !hold.Taken() {
		t.Errorf("hold = %+v", hold)
	}
	if HoldOf(Actor{Name: "dbp-tl"}).Taken() {
		t.Error("un actor sin usuario no toma nada")
	}
}

func TestHeldByOther(t *testing.T) {
	ana, bea := uuid.New(), uuid.New()
	hold := Hold{HolderID: ana, TakenBy: "ana"}
	if hold.HeldByOther(Actor{UserID: ana, Profile: ProfileTL}) {
		t.Error("cualquier perfil de la misma persona trabaja en lo suyo")
	}
	if !hold.HeldByOther(Actor{UserID: bea}) {
		t.Error("otra persona no escribe")
	}
	if (Hold{}).HeldByOther(Actor{UserID: bea}) {
		t.Error("libre no bloquea a nadie")
	}
}
