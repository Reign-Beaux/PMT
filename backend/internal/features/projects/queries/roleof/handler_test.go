package roleof

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

type fakeReader struct {
	kind string
	err  error
}

func (r *fakeReader) RoleOf(_ context.Context, _ uuid.UUID, kind string, _ uuid.UUID) (uuid.UUID, domain.Role, bool, error) {
	r.kind = kind
	return uuid.Nil, domain.RoleViewer, true, r.err
}

func TestHandle(t *testing.T) {
	reader := &fakeReader{}

	result, err := New(reader).Handle(context.Background(), uuid.New(), "task", uuid.New())

	if err != nil || !result.Found || result.Role != domain.RoleViewer || reader.kind != "task" {
		t.Fatalf("result = %+v, kind = %q, err = %v", result, reader.kind, err)
	}
}

func TestHandleKeepsErrors(t *testing.T) {
	want := errors.New("sin base")

	if _, err := New(&fakeReader{err: want}).Handle(context.Background(), uuid.New(), "stage", uuid.New()); !errors.Is(err, want) {
		t.Errorf("err = %v", err)
	}
}
