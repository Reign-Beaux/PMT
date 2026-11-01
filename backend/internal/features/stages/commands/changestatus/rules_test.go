package changestatus

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

func TestHandleRule4(t *testing.T) {
	empty := kernel.Conflict("la etapa no tiene tareas activas")

	t.Run("con tarea activa", func(t *testing.T) {
		repo, id := newRepoWith(t, domain.StatusPlanning)
		tasks := &fakeTasks{}

		if _, err := New(repo, tasks, waitingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: kernel.Owner}); err != nil {
			t.Fatal(err)
		}
		if tasks.checks != 1 {
			t.Errorf("consultas = %d", tasks.checks)
		}
	})

	t.Run("sin tarea activa", func(t *testing.T) {
		repo, id := newRepoWith(t, domain.StatusPlanning)

		_, err := New(repo, &fakeTasks{releasable: empty}, waitingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: kernel.Owner})

		if !errors.Is(err, empty) {
			t.Fatalf("err = %v", err)
		}
		if repo.updates != 0 {
			t.Error("no debió guardar")
		}
	})

	t.Run("solo desde planning", func(t *testing.T) {
		repo, id := newRepoWith(t, domain.StatusStopped)

		if _, err := New(repo, &fakeTasks{releasable: empty}, waitingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: kernel.Owner}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("planning propio sin tarea activa", func(t *testing.T) {
		s, err := domain.NewOwnPlanningStage(uuid.New(), "Pagos", "Inicial", "Petición", "PM")
		if err != nil {
			t.Fatal(err)
		}
		repo := &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}

		_, err = New(repo, &fakeTasks{releasable: empty}, workingProject).Handle(context.Background(), Command{ID: s.ID, Status: domain.StatusStandBy, Actor: kernel.Owner})

		if !errors.Is(err, empty) {
			t.Fatalf("err = %v", err)
		}
		if len(repo.changes) != 0 || !repo.stages[s.ID].InOwnPlanning() {
			t.Errorf("no debió salir: %+v", repo.stages[s.ID])
		}
	})

	t.Run("planning propio a detenida no la exige", func(t *testing.T) {
		s, err := domain.NewOwnPlanningStage(uuid.New(), "Pagos", "Inicial", "Petición", "PM")
		if err != nil {
			t.Fatal(err)
		}
		repo := &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}

		if _, err := New(repo, &fakeTasks{releasable: empty}, workingProject).Handle(context.Background(), Command{ID: s.ID, Status: domain.StatusStopped, Actor: kernel.Owner, Reason: "motivo"}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestHandleRule5(t *testing.T) {
	for _, from := range []domain.Status{domain.StatusStandBy, domain.StatusStopped, domain.StatusDone} {
		t.Run(string(from)+" con proyecto en trabajo", func(t *testing.T) {
			repo, id := newRepoWith(t, from)

			result, err := New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusInDev, Actor: actorFor(domain.StatusInDev), Reason: "motivo"})

			if err != nil {
				t.Fatal(err)
			}
			if result.Status != domain.StatusInDev {
				t.Errorf("result = %+v", result)
			}
		})

		t.Run(string(from)+" con proyecto sin iniciar", func(t *testing.T) {
			repo, id := newRepoWith(t, from)

			_, err := New(repo, &fakeTasks{}, waitingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusInDev, Actor: actorFor(domain.StatusInDev), Reason: "motivo"})

			if !errors.Is(err, domain.ErrStageProjectNotStarted) {
				t.Fatalf("err = %v", err)
			}
			if repo.updates != 0 || repo.stages[id].Status != from {
				t.Errorf("no debió guardar: %+v", repo.stages[id])
			}
		})
	}
}

func TestHandleRule10(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		to   domain.Status
		want error
	}{
		{"detener", domain.StatusInDev, domain.StatusStopped, nil},
		{"cancelar", domain.StatusStopped, domain.StatusCancelled, nil},
		{"liberar", domain.StatusPlanning, domain.StatusStandBy, domain.ErrStageProjectFrozen},
		{"devolver", domain.StatusStandBy, domain.StatusPlanning, domain.ErrStageProjectFrozen},
		{"reanudar", domain.StatusStopped, domain.StatusInDev, domain.ErrStageProjectFrozen},
		{"cerrar", domain.StatusInDev, domain.StatusDone, domain.ErrStageProjectFrozen},
		{"reabrir", domain.StatusCancelled, domain.StatusPlanning, domain.ErrStageProjectFrozen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)
			tasks := &fakeTasks{allDone: true}

			_, err := New(repo, tasks, frozenProject).Handle(context.Background(), Command{ID: id, Status: tt.to, Actor: actorFor(tt.to), Reason: "motivo"})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if tt.want != nil && repo.updates != 0 {
				t.Error("no debió guardar")
			}
		})
	}
}

func TestHandleRule10InOwnPlanning(t *testing.T) {
	s, err := domain.NewOwnPlanningStage(uuid.New(), "Pagos", "Inicial", "Petición", "PM")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}

	_, err = New(repo, &fakeTasks{}, frozenProject).Handle(context.Background(), Command{ID: s.ID, Status: domain.StatusStandBy, Actor: kernel.Owner})

	if !errors.Is(err, domain.ErrStageProjectFrozen) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.changes) != 0 {
		t.Errorf("cambios = %+v", repo.changes)
	}
}

func TestHandleProjectCheckFailure(t *testing.T) {
	failure := errors.New("fallo de lectura")
	repo, id := newRepoWith(t, domain.StatusPlanning)

	_, err := New(repo, &fakeTasks{}, fakeProjects{err: failure}).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: kernel.Owner})

	if !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}

// Pendiente de la fase 3: User no aprueba por TL, aunque la etapa pueda cerrarse.
func TestHandleUserCannotCloseStageWithAllTasksDone(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusInDev)
	tasks := &fakeTasks{allDone: true}

	_, err := New(repo, tasks, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusDone, Actor: kernel.Owner})

	if !errors.Is(err, kernel.ErrTransitionForbidden) {
		t.Fatalf("err = %v", err)
	}
	if repo.updates != 0 || repo.stages[id].Status != domain.StatusInDev || tasks.calls != 0 {
		t.Errorf("updates = %d, etapa = %+v, consultas = %d", repo.updates, repo.stages[id], tasks.calls)
	}
}
