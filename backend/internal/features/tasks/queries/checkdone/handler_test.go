package checkdone

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
		want  domain.Closure
	}{
		{"sin tareas", nil, domain.Closure{}},
		{"todas en done", []domain.Task{{Status: domain.StatusDone}, {Status: domain.StatusDone}}, domain.Closure{Done: 2}},
		{"alguna sin terminar", []domain.Task{{Status: domain.StatusDone}, {Status: domain.StatusBacklog}}, domain.Closure{Done: 1, Pending: 1}},
		{"casi cerrada", []domain.Task{{Status: domain.StatusPendingClose}}, domain.Closure{Pending: 1}},
		{"cancelada no cuenta", []domain.Task{{Status: domain.StatusDone}, {Status: domain.StatusCancelled}}, domain.Closure{Done: 1}},
		{"toda cancelada", []domain.Task{{Status: domain.StatusCancelled}}, domain.Closure{}},
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

	got, err := New(&fakeReader{err: failure}).Handle(context.Background(), uuid.New())

	if !errors.Is(err, failure) || got != (domain.Closure{}) {
		t.Fatalf("got = %v, err = %v", got, err)
	}
}
