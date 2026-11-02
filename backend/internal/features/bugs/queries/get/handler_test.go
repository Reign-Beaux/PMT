package get

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

type fakeReader struct {
	existing uuid.UUID
}

func (r fakeReader) Get(_ context.Context, id uuid.UUID) (domain.Bug, error) {
	if id != r.existing {
		return domain.Bug{}, domain.ErrBugNotFound
	}
	return domain.Bug{ID: id}, nil
}

func TestHandle(t *testing.T) {
	id := uuid.New()
	handler := New(fakeReader{existing: id})

	if err := handler.Handle(context.Background(), id); err != nil {
		t.Error(err)
	}
	if err := handler.Handle(context.Background(), uuid.New()); !errors.Is(err, domain.ErrBugNotFound) {
		t.Errorf("err = %v", err)
	}
}
