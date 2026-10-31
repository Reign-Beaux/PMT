package checktasks

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

var (
	errNotPlanning = kernel.Conflict("el proyecto no está en planning")
	errNotWorking  = kernel.Conflict("el proyecto no está en working")
)

type fakeReader struct {
	stage  domain.Stage
	hasAdd bool
	err    error
}

func (r fakeReader) Get(context.Context, uuid.UUID) (domain.Stage, error) {
	return r.stage, r.err
}

func (r fakeReader) HasAdd(context.Context, uuid.UUID) (bool, error) {
	return r.hasAdd, nil
}

type fakeProjects struct {
	planning error
	working  error
}

func (p fakeProjects) EnsurePlanning(context.Context, uuid.UUID) error {
	return p.planning
}

func (p fakeProjects) EnsureWorking(context.Context, uuid.UUID) error {
	return p.working
}

func TestEnsureEditable(t *testing.T) {
	ownPlanning := domain.Stage{AddReason: "Petición", AddRequestedBy: "PM"}
	inWorking := fakeProjects{planning: errNotPlanning}
	inPlanning := fakeProjects{working: errNotWorking}
	stopped := fakeProjects{planning: errNotPlanning, working: errNotWorking}

	tests := []struct {
		name     string
		reader   fakeReader
		projects fakeProjects
		want     error
	}{
		{"planning propio en working", fakeReader{stage: ownPlanning}, inWorking, nil},
		{"planning propio fuera de working", fakeReader{stage: ownPlanning}, stopped, errNotWorking},
		{"planning propio con proyecto en planning", fakeReader{stage: ownPlanning}, inPlanning, errNotWorking},
		{"salió de su planning propio, en working", fakeReader{hasAdd: true}, inWorking, errNotPlanning},
		{"salió y el proyecto volvió a planning", fakeReader{hasAdd: true}, inPlanning, domain.ErrStageHasChanges},
		{"definida en planning, proyecto en planning", fakeReader{}, inPlanning, nil},
		{"definida en planning, proyecto en working", fakeReader{}, inWorking, errNotPlanning},
		{"no existe", fakeReader{err: domain.ErrStageNotFound}, inWorking, domain.ErrStageNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader, tt.projects).EnsureEditable(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestEnsureHistoryRule(t *testing.T) {
	tests := []struct {
		name   string
		reader fakeReader
		want   error
	}{
		{"planning propio", fakeReader{stage: domain.Stage{AddReason: "Petición", AddRequestedBy: "PM"}}, domain.ErrStageInOwnPlanning},
		{"otra etapa", fakeReader{}, nil},
		{"no existe", fakeReader{err: domain.ErrStageNotFound}, domain.ErrStageNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.reader, fakeProjects{}).EnsureHistoryRule(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v", err)
			}
		})
	}
}
