package checkprogress

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

type fakeReader struct {
	stages     []domain.Stage
	versionOne []domain.Stage
	err        error
}

func (r fakeReader) List(context.Context, uuid.UUID) ([]domain.Stage, error) {
	return r.stages, r.err
}

func (r fakeReader) ListVersionOne(context.Context, uuid.UUID) ([]domain.Stage, error) {
	return r.versionOne, r.err
}

func withStatus(statuses ...domain.Status) fakeReader {
	reader := fakeReader{}
	for _, status := range statuses {
		reader.stages = append(reader.stages, domain.Stage{Status: status})
	}
	reader.versionOne = reader.stages
	return reader
}

func TestEnsureReleasable(t *testing.T) {
	tests := []struct {
		name   string
		reader fakeReader
		want   error
	}{
		{"una en stand_by", withStatus(domain.StatusStandBy), nil},
		{"una detenida cuenta como activa", withStatus(domain.StatusStopped), nil},
		{"canceladas y una activa", withStatus(domain.StatusCancelled, domain.StatusStandBy), nil},
		{"sin etapas", withStatus(), domain.ErrNoActiveStages},
		{"solo canceladas", withStatus(domain.StatusCancelled), domain.ErrNoActiveStages},
		{"una en planning", withStatus(domain.StatusStandBy, domain.StatusPlanning), domain.ErrStagesInPlanning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader).EnsureReleasable(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestEnsureStartable(t *testing.T) {
	tests := []struct {
		name   string
		reader fakeReader
		want   error
	}{
		{"una en stand_by", withStatus(domain.StatusPlanning, domain.StatusStandBy), nil},
		{"sin etapas", withStatus(), domain.ErrNoStagesStandBy},
		{"ninguna en stand_by", withStatus(domain.StatusPlanning, domain.StatusDone), domain.ErrNoStagesStandBy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader).EnsureStartable(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestEnsureClosed(t *testing.T) {
	tests := []struct {
		name   string
		reader fakeReader
		want   error
	}{
		{"todas en done", withStatus(domain.StatusDone, domain.StatusDone), nil},
		{"done y cancelada", withStatus(domain.StatusDone, domain.StatusCancelled), nil},
		{"sin etapas", withStatus(), domain.ErrNoStagesDone},
		{"solo canceladas", withStatus(domain.StatusCancelled), domain.ErrNoStagesDone},
		{"una en in_dev", withStatus(domain.StatusDone, domain.StatusInDev), domain.ErrStagesOpen},
		{"una detenida", withStatus(domain.StatusDone, domain.StatusStopped), domain.ErrStagesOpen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader).EnsureClosed(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestEnsureClosedOnlyCountsVersionOne(t *testing.T) {
	added := func(status domain.Status, versionOne ...domain.Status) fakeReader {
		reader := withStatus(versionOne...)
		reader.stages = append(reader.stages, domain.Stage{Status: status})
		return reader
	}
	tests := []struct {
		name   string
		reader fakeReader
		want   error
	}{
		{"versión 1 terminada y agregada abierta", added(domain.StatusInDev, domain.StatusDone, domain.StatusCancelled), nil},
		{"versión 1 abierta y agregada en done", added(domain.StatusDone, domain.StatusDone, domain.StatusInDev), domain.ErrStagesOpen},
		{"versión 1 cancelada y agregada en done", added(domain.StatusDone, domain.StatusCancelled), domain.ErrNoStagesDone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader).EnsureClosed(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestReadFailure(t *testing.T) {
	failure := errors.New("fallo de lectura")

	if err := New(fakeReader{err: failure}).EnsureClosed(context.Background(), uuid.New()); !errors.Is(err, failure) {
		t.Errorf("err = %v", err)
	}
}
