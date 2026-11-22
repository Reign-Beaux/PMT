package instanceowner

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
)

type fakeReader struct {
	user domain.User
	err  error
}

func (r fakeReader) InstanceOwner(context.Context) (domain.User, error) {
	return r.user, r.err
}

func TestHandle(t *testing.T) {
	id := uuid.New()
	if got, err := New(fakeReader{user: domain.User{ID: id}}).Handle(context.Background()); err != nil || got != id {
		t.Fatalf("id = %v, err = %v", got, err)
	}
}

func TestHandleWithoutOwner(t *testing.T) {
	if got, err := New(fakeReader{err: domain.ErrUserNotFound}).Handle(context.Background()); err != nil || got != uuid.Nil {
		t.Fatalf("id = %v, err = %v", got, err)
	}
}

func TestHandleKeepsErrors(t *testing.T) {
	want := errors.New("sin base")
	if _, err := New(fakeReader{err: want}).Handle(context.Background()); !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}
