package delete

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

var errNotPlanning = kernel.Conflict("el proyecto no está en planning")

type fakeStages struct {
	err error
}

func (s fakeStages) EnsureProjectPlanning(context.Context, uuid.UUID) error {
	return s.err
}

type fakeRepo struct {
	hasChanges bool
	existing   uuid.UUID
	deleted    bool
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	if id != r.existing {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return domain.Task{ID: id}, nil
}

func (r *fakeRepo) Delete(context.Context, uuid.UUID) error {
	r.deleted = true
	return nil
}

func (r *fakeRepo) HasChanges(context.Context, uuid.UUID) (bool, error) {
	return r.hasChanges, nil
}

func TestHandleDeletesInPlanning(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id}

	if err := New(repo, fakeStages{}).Handle(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if !repo.deleted {
		t.Error("debió eliminar la tarea")
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id}

	err := New(repo, fakeStages{err: errNotPlanning}).Handle(context.Background(), id)

	if !errors.Is(err, errNotPlanning) {
		t.Fatalf("err = %v", err)
	}
	if repo.deleted {
		t.Error("no debió eliminar la tarea")
	}
}

func TestHandleRejectsElementWithChanges(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id, hasChanges: true}

	err := New(repo, fakeStages{}).Handle(context.Background(), id)

	if !errors.Is(err, domain.ErrTaskHasChanges) {
		t.Fatalf("err = %v", err)
	}
	if repo.deleted {
		t.Error("no debió eliminar")
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name   string
		id     string
		stages error
		status int
	}{
		{"existe", id.String(), nil, http.StatusNoContent},
		{"id inválido", "abc", nil, http.StatusBadRequest},
		{"no existe", uuid.NewString(), nil, http.StatusNotFound},
		{"fuera de planning", id.String(), errNotPlanning, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{existing: id}, fakeStages{err: tt.stages}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/tasks/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
