package holdof

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

type fakeReader struct {
	kind string
	hold kernel.Hold
	err  error
}

func (f fakeReader) HoldOf(context.Context, string, uuid.UUID) (string, kernel.Hold, error) {
	return f.kind, f.hold, f.err
}

func TestHandle(t *testing.T) {
	hold := kernel.Hold{HolderID: uuid.New(), TakenBy: "ana"}
	got, err := New(fakeReader{kind: "stage", hold: hold}).Handle(context.Background(), "task", uuid.New())
	if err != nil || got.HolderKind != "stage" || got.Hold != hold {
		t.Fatalf("got = %+v, err = %v", got, err)
	}
}

func TestHandleKeepsErrors(t *testing.T) {
	want := errors.New("sin base")
	if _, err := New(fakeReader{err: want}).Handle(context.Background(), "stage", uuid.New()); !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}
