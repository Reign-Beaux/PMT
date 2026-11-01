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
	accepts  bool
	frozen   bool
}

func (p fakeProjects) EnsurePlanning(context.Context, uuid.UUID) error {
	return p.planning
}

func (p fakeProjects) EnsureWorking(context.Context, uuid.UUID) error {
	return p.working
}

func (p fakeProjects) AcceptsWork(context.Context, uuid.UUID) (bool, error) {
	return p.accepts, nil
}

func (p fakeProjects) IsFrozen(context.Context, uuid.UUID) (bool, error) {
	return p.frozen, nil
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

func TestEnsureWorkOpen(t *testing.T) {
	open := fakeProjects{accepts: true}
	waiting := fakeProjects{}
	frozen := fakeProjects{frozen: true}

	tests := []struct {
		name     string
		stage    domain.Status
		projects fakeProjects
		want     error
	}{
		{"regla 1: proyecto y etapa en trabajo", domain.StatusInDev, open, nil},
		{"regla 1: proyecto sin iniciar", domain.StatusInDev, waiting, domain.ErrWorkProjectClosed},
		{"regla 1: etapa en stand_by", domain.StatusStandBy, open, domain.ErrWorkStageClosed},
		{"regla 1: etapa en done", domain.StatusDone, open, domain.ErrWorkStageClosed},
		{"regla 7: etapa detenida", domain.StatusStopped, open, domain.ErrWorkStageClosed},
		{"regla 8: etapa cancelada", domain.StatusCancelled, open, domain.ErrWorkStageClosed},
		{"regla 9: proyecto detenido o cancelado", domain.StatusInDev, frozen, domain.ErrWorkProjectClosed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := fakeReader{stage: domain.Stage{Status: tt.stage}}
			err := New(reader, tt.projects).EnsureWorkOpen(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestEnsureTaskCancellable(t *testing.T) {
	waiting := fakeProjects{}
	frozen := fakeProjects{frozen: true}

	tests := []struct {
		name     string
		stage    domain.Status
		projects fakeProjects
		want     error
	}{
		{"reglas 2 y 3: etapa en in_dev", domain.StatusInDev, fakeProjects{accepts: true}, nil},
		{"reglas 2 y 3: proyecto en planning", domain.StatusPlanning, waiting, nil},
		{"etapa detenida sí admite", domain.StatusStopped, waiting, nil},
		{"reglas 2 y 3: etapa en done", domain.StatusDone, waiting, domain.ErrTasksFrozenByStage},
		{"regla 8: etapa cancelada", domain.StatusCancelled, waiting, domain.ErrTasksFrozenByStage},
		{"regla 9: proyecto detenido o cancelado", domain.StatusInDev, frozen, domain.ErrTasksFrozenByProject},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := fakeReader{stage: domain.Stage{Status: tt.stage}}
			err := New(reader, tt.projects).EnsureTaskCancellable(context.Background(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, se esperaba %v", err, tt.want)
			}
		})
	}
}

func TestRulesReturnNotFound(t *testing.T) {
	h := New(fakeReader{err: domain.ErrStageNotFound}, fakeProjects{})

	if err := h.EnsureWorkOpen(context.Background(), uuid.New()); !errors.Is(err, domain.ErrStageNotFound) {
		t.Errorf("EnsureWorkOpen: err = %v", err)
	}
	if err := h.EnsureTaskCancellable(context.Background(), uuid.New()); !errors.Is(err, domain.ErrStageNotFound) {
		t.Errorf("EnsureTaskCancellable: err = %v", err)
	}
}
