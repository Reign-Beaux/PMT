package delete

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

var errNotPlanning = kernel.Conflict("el proyecto no está en planning")

type fakeProjects struct {
	err     error
	working error
}

func (p fakeProjects) EnsurePlanning(context.Context, uuid.UUID) error {
	return p.err
}

func (p fakeProjects) EnsureWorking(context.Context, uuid.UUID) error {
	return p.working
}

type fakeRepo struct {
	hasChanges bool
	ownPlan    bool
	existing   uuid.UUID
	err        error
	deleted    bool
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	if id != r.existing {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	if r.ownPlan {
		return domain.Stage{ID: id, AddReason: "Petición", AddRequestedBy: "PM"}, nil
	}
	return domain.Stage{ID: id}, nil
}

func (r *fakeRepo) Delete(context.Context, uuid.UUID) error {
	if r.err != nil {
		return r.err
	}
	r.deleted = true
	return nil
}

func (r *fakeRepo) HasChanges(context.Context, uuid.UUID) (bool, error) {
	return r.hasChanges, nil
}

func TestHandleDeletesInPlanning(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id}

	if err := New(repo, fakeProjects{}).Handle(context.Background(), id, kernel.Owner); err != nil {
		t.Fatal(err)
	}
	if !repo.deleted {
		t.Error("debió eliminar la etapa")
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id}

	err := New(repo, fakeProjects{err: errNotPlanning}).Handle(context.Background(), id, kernel.Owner)

	if !errors.Is(err, errNotPlanning) {
		t.Fatalf("err = %v", err)
	}
	if repo.deleted {
		t.Error("no debió eliminar la etapa")
	}
}

func TestHandleRejectsElementWithChanges(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id, hasChanges: true}

	err := New(repo, fakeProjects{}).Handle(context.Background(), id, kernel.Owner)

	if !errors.Is(err, domain.ErrStageHasChanges) {
		t.Fatalf("err = %v", err)
	}
	if repo.deleted {
		t.Error("no debió eliminar")
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name     string
		id       string
		err      error
		projects error
		status   int
	}{
		{"existe", id.String(), nil, nil, http.StatusNoContent},
		{"id inválido", "abc", nil, nil, http.StatusBadRequest},
		{"no existe", uuid.NewString(), nil, nil, http.StatusNotFound},
		{"con tareas", id.String(), domain.ErrStageHasTasks, nil, http.StatusConflict},
		{"solo con tareas retiradas", id.String(), domain.ErrStageHasTasks, nil, http.StatusConflict},
		{"fuera de planning", id.String(), nil, errNotPlanning, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{existing: id, err: tt.err}, fakeProjects{err: tt.projects}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/stages/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandleDeletesStageInOwnPlanning(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id, ownPlan: true}

	err := New(repo, fakeProjects{err: errNotPlanning}).Handle(context.Background(), id, kernel.Owner)

	if err != nil {
		t.Fatal(err)
	}
	if !repo.deleted {
		t.Error("no eliminó la etapa")
	}
}

func TestHandleRejectsOwnPlanningOutsideWorking(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id, ownPlan: true}
	errNotWorking := kernel.Conflict("el proyecto no está en working")

	err := New(repo, fakeProjects{working: errNotWorking}).Handle(context.Background(), id, kernel.Owner)

	if !errors.Is(err, errNotWorking) || repo.deleted {
		t.Errorf("err = %v, eliminada = %v", err, repo.deleted)
	}
}
