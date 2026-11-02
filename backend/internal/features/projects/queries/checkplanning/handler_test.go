package checkplanning

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
		name   string
		reader fakeReader
		want   error
	}{
		{"en planning", fakeReader{project: domain.Project{Status: domain.StatusPlanning}}, nil},
		{"en stand_by", fakeReader{project: domain.Project{Status: domain.StatusStandBy}}, domain.ErrProjectNotPlanning},
		{"no existe", fakeReader{err: domain.ErrProjectNotFound}, domain.ErrProjectNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := New(tt.reader).Handle(context.Background(), uuid.New()); !errors.Is(err, tt.want) {
				t.Errorf("err = %v", err)
			}
		})
	}
}
