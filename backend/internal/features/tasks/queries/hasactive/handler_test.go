package hasactive

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

type fakeReader struct {
	tasks []domain.Task
	err   error
}

func (r *fakeReader) List(context.Context, uuid.UUID) ([]domain.Task, error) {
	return r.tasks, r.err
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name  string
		tasks []domain.Task
		want  bool
	}{
		{"sin tareas", nil, false},
		{"con tareas", []domain.Task{{Status: domain.StatusDone}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(&fakeReader{tasks: tt.tasks}).Handle(context.Background(), uuid.New())

			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got = %v", got)
			}
		})
	}
}

func TestHandleReturnsReadError(t *testing.T) {
	failure := errors.New("fallo de lectura")

	if _, err := New(&fakeReader{err: failure}).Handle(context.Background(), uuid.New()); !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}
