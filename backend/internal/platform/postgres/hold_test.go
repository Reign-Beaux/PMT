package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/kernel"
)

func TestHoldOf(t *testing.T) {
	id, at := uuid.New(), time.Now()
	got := HoldOf(pgtype.UUID{Bytes: id, Valid: true}, pgtype.Text{String: "ana", Valid: true}, pgtype.Timestamptz{Time: at, Valid: true})
	if got != (kernel.Hold{HolderID: id, TakenBy: "ana", TakenAt: at}) {
		t.Errorf("tomada = %+v", got)
	}
	if got := HoldOf(pgtype.UUID{}, pgtype.Text{}, pgtype.Timestamptz{}); got.Taken() {
		t.Errorf("libre = %+v", got)
	}
}

func TestHoldParams(t *testing.T) {
	id := uuid.New()
	holder, name := HoldParams(kernel.Hold{HolderID: id, TakenBy: "ana"})
	if !holder.Valid || uuid.UUID(holder.Bytes) != id || name != (pgtype.Text{String: "ana", Valid: true}) {
		t.Errorf("tomada = %v %v", holder, name)
	}
	if holder, name := HoldParams(kernel.Hold{}); holder.Valid || name.Valid {
		t.Errorf("libre = %v %v", holder, name)
	}
}
