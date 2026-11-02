package checkrelease

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

func (r fakeReader) List(context.Context, uuid.UUID) ([]domain.Task, error) {
	return r.tasks, r.err
}

func TestHandle(t *testing.T) {
	failure := errors.New("fallo de lectura")
	tests := []struct {
		name   string
		reader fakeReader
		want   error
	}{
		{"una activa", fakeReader{tasks: []domain.Task{{Status: domain.StatusCancelled}, {Status: domain.StatusBacklog}}}, nil},
		{"sin tareas", fakeReader{}, domain.ErrStageNoActiveTasks},
		{"solo canceladas", fakeReader{tasks: []domain.Task{{Status: domain.StatusCancelled}}}, domain.ErrStageNoActiveTasks},
		{"fallo de lectura", fakeReader{err: failure}, failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader).Handle(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}
