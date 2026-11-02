package checkworking

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

type fakeReader struct {
	project domain.Project
	err     error
}

func (r fakeReader) Get(context.Context, uuid.UUID) (domain.Project, error) {
	return r.project, r.err
}

func TestHandle(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   bool
	}{
		{domain.StatusWorking, true},
		{domain.StatusInDev, false},
		{domain.StatusPlanning, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got, err := New(fakeReader{project: domain.Project{Status: tt.status}}).Handle(context.Background(), uuid.New())
			if err != nil || got != tt.want {
				t.Errorf("got = %v, err = %v", got, err)
			}
		})
	}
}

func TestHandleReturnsNotFound(t *testing.T) {
	_, err := New(fakeReader{err: domain.ErrProjectNotFound}).Handle(context.Background(), uuid.New())

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("err = %v", err)
	}
}
