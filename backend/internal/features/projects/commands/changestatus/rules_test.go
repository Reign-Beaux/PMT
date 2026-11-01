package changestatus

import (
	"context"
	"errors"
	"testing"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

func TestHandleStageRules(t *testing.T) {
	blocked := kernel.Conflict("regla")
	tests := []struct {
		name    string
		from    domain.Status
		to      domain.Status
		reached bool
		stages  fakeStages
		want    error
		calls   int
	}{
		{"regla 11 permitida", domain.StatusPlanning, domain.StatusStandBy, false, fakeStages{}, nil, 1},
		{"regla 11 rechazada", domain.StatusPlanning, domain.StatusStandBy, false, fakeStages{releasable: blocked}, blocked, 1},
		{"regla 11 solo desde planning", domain.StatusStopped, domain.StatusStandBy, false, fakeStages{releasable: blocked}, nil, 0},
		{"regla 12 permitida", domain.StatusStandBy, domain.StatusInDev, false, fakeStages{}, nil, 1},
		{"regla 12 rechazada", domain.StatusStandBy, domain.StatusInDev, false, fakeStages{startable: blocked}, blocked, 1},
		{"regla 12 solo desde stand_by", domain.StatusStopped, domain.StatusInDev, false, fakeStages{startable: blocked}, nil, 0},
		{"in_dev a working ya no es manual", domain.StatusInDev, domain.StatusWorking, false, fakeStages{}, domain.ErrProjectStatusTransition, 0},
		{"stopped sin haber estado en working, permitida", domain.StatusStopped, domain.StatusWorking, false, fakeStages{}, nil, 1},
		{"stopped sin haber estado en working, rechazada", domain.StatusStopped, domain.StatusWorking, false, fakeStages{closed: blocked}, blocked, 1},
		{"stopped habiendo estado en working no comprueba", domain.StatusStopped, domain.StatusWorking, true, fakeStages{closed: blocked}, nil, 0},
		{"detener no consulta etapas", domain.StatusInDev, domain.StatusStopped, false, fakeStages{closed: blocked}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)
			p := repo.projects[id]
			p.Status = tt.from
			repo.projects[id] = p
			repo.reached = tt.reached
			stages := tt.stages

			_, err := New(repo, &stages).Handle(context.Background(), Command{ID: id, Status: tt.to, Actor: kernel.Owner, Reason: "motivo"})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if stages.calls != tt.calls {
				t.Errorf("consultas = %d, se esperaban %d", stages.calls, tt.calls)
			}
			if tt.want != nil && (repo.updates != 0 || repo.projects[id].Status != tt.from) {
				t.Errorf("no debió guardar: %+v", repo.projects[id])
			}
		})
	}
}

func TestHandleProfileFailsBeforeCheckingStages(t *testing.T) {
	repo, id := newRepoWith(t)
	stages := &fakeStages{}

	_, err := New(repo, stages).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}})

	if !errors.Is(err, kernel.ErrTransitionForbidden) {
		t.Fatalf("err = %v", err)
	}
	if stages.calls != 0 {
		t.Errorf("consultas = %d", stages.calls)
	}
}
