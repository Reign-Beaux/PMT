package checkstatus

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

func TestStatus(t *testing.T) {
	tests := []struct {
		status  domain.Status
		accepts bool
		frozen  bool
	}{
		{domain.StatusPlanning, false, false},
		{domain.StatusStandBy, false, false},
		{domain.StatusInDev, true, false},
		{domain.StatusWorking, true, false},
		{domain.StatusStopped, false, true},
		{domain.StatusCancelled, false, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			h := New(fakeReader{project: domain.Project{Status: tt.status}})

			accepts, err := h.AcceptsWork(context.Background(), uuid.New())
			if err != nil || accepts != tt.accepts {
				t.Errorf("AcceptsWork = %v, err = %v", accepts, err)
			}
			frozen, err := h.IsFrozen(context.Background(), uuid.New())
			if err != nil || frozen != tt.frozen {
				t.Errorf("IsFrozen = %v, err = %v", frozen, err)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	h := New(fakeReader{err: domain.ErrProjectNotFound})

	if _, err := h.AcceptsWork(context.Background(), uuid.New()); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("AcceptsWork: err = %v", err)
	}
	if _, err := h.IsFrozen(context.Background(), uuid.New()); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("IsFrozen: err = %v", err)
	}
}
