package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

var errNotPlanning = kernel.Conflict("el proyecto no está en planning")

type fakeStages struct {
	err error
}

func (s fakeStages) EnsureTasksEditable(context.Context, uuid.UUID) error {
	return s.err
}

type fakeRepo struct {
	tasks      map[uuid.UUID]domain.Task
	hasChanges bool
}

func newRepoWith(t *testing.T) (*fakeRepo, uuid.UUID) {
	t.Helper()
	task, err := domain.NewTask(uuid.New(), "Revisar", "Detalle")
	if err != nil {
		t.Fatal(err)
	}
	return &fakeRepo{tasks: map[uuid.UUID]domain.Task{task.ID: task}}, task.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	task, ok := r.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return task, nil
}

func (r *fakeRepo) Update(_ context.Context, task domain.Task) (domain.Task, error) {
	r.tasks[task.ID] = task
	return task, nil
}

func (r *fakeRepo) HasChanges(context.Context, uuid.UUID) (bool, error) {
	return r.hasChanges, nil
}

func TestHandleUpdatesTask(t *testing.T) {
	repo, id := newRepoWith(t)

	result, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Title: "Probar", Description: "Otra"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Probar" || repo.tasks[id].Description != "Otra" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleKeepsTaskWhenInvalid(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Title: " "})

	if !errors.Is(err, domain.ErrTaskTitleRequired) {
		t.Fatalf("err = %v", err)
	}
	if repo.tasks[id].Title != "Revisar" {
		t.Errorf("tarea = %+v", repo.tasks[id])
	}
}

func TestHandleLastWriteWins(t *testing.T) {
	repo, id := newRepoWith(t)
	handler := New(repo, fakeStages{})
	if _, err := handler.Handle(context.Background(), Command{ID: id, Title: "Probar"}); err != nil {
		t.Fatal(err)
	}

	if _, err := handler.Handle(context.Background(), Command{ID: id, Title: "Tarde"}); err != nil {
		t.Fatal(err)
	}

	if repo.tasks[id].Title != "Tarde" {
		t.Errorf("tarea = %+v", repo.tasks[id])
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo, fakeStages{err: errNotPlanning}).Handle(context.Background(), Command{ID: id, Title: "Otra"})

	if !errors.Is(err, errNotPlanning) {
		t.Fatalf("err = %v", err)
	}
	if repo.tasks[id].Title != "Revisar" {
		t.Errorf("tarea = %+v", repo.tasks[id])
	}
}

func TestHandleRejectsElementWithChanges(t *testing.T) {
	repo, id := newRepoWith(t)
	repo.hasChanges = true
	before := repo.tasks[id].Title

	_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Title: "Otra"})

	if !errors.Is(err, domain.ErrTaskHasChanges) {
		t.Fatalf("err = %v", err)
	}
	if repo.tasks[id].Title != before {
		t.Errorf("elemento = %+v", repo.tasks[id])
	}
}

func TestRESTRejectsOutsidePlanning(t *testing.T) {
	repo, id := newRepoWith(t)
	mux := http.NewServeMux()
	New(repo, fakeStages{err: errNotPlanning}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/tasks/"+id.String(), strings.NewReader(`{"title":"Otra"}`)))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name   string
		id     func(uuid.UUID) string
		body   string
		status int
	}{
		{"válida", uuid.UUID.String, `{"title":"Probar"}`, http.StatusOK},
		{"expected_version desconocido", uuid.UUID.String, `{"title":"Probar","expected_version":1}`, http.StatusBadRequest},
		{"id inválido", func(uuid.UUID) string { return "abc" }, `{"title":"A"}`, http.StatusBadRequest},
		{"no existe", func(uuid.UUID) string { return uuid.NewString() }, `{"title":"A"}`, http.StatusNotFound},
		{"título vacío", uuid.UUID.String, `{"title":""}`, http.StatusBadRequest},
		{"status no editable", uuid.UUID.String, `{"title":"A","status":"ready"}`, http.StatusBadRequest},
		{"blocked desconocido", uuid.UUID.String, `{"title":"A","blocked":true}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)
			mux := http.NewServeMux()
			New(repo, fakeStages{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/tasks/"+tt.id(id), strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
