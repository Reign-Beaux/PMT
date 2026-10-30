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
)

type fakeRepo struct {
	tasks map[uuid.UUID]domain.Task
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

func (r *fakeRepo) Update(_ context.Context, task domain.Task, expectedVersion int64) (domain.Task, error) {
	if r.tasks[task.ID].Version != expectedVersion {
		return domain.Task{}, domain.ErrTaskVersionConflict
	}
	task.Version++
	r.tasks[task.ID] = task
	return task, nil
}

func TestHandleUpdatesTask(t *testing.T) {
	repo, id := newRepoWith(t)

	result, err := New(repo).Handle(context.Background(), Command{ID: id, Title: "Probar", Description: "Otra", ExpectedVersion: 1})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Probar" || result.Version != 2 || repo.tasks[id].Description != "Otra" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleKeepsTaskWhenInvalid(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo).Handle(context.Background(), Command{ID: id, Title: " ", ExpectedVersion: 1})

	if !errors.Is(err, domain.ErrTaskTitleRequired) {
		t.Fatalf("err = %v", err)
	}
	if repo.tasks[id].Title != "Revisar" {
		t.Errorf("tarea = %+v", repo.tasks[id])
	}
}

func TestHandleRejectsStaleVersion(t *testing.T) {
	repo, id := newRepoWith(t)
	handler := New(repo)
	cmd := Command{ID: id, Title: "Probar", ExpectedVersion: 1}
	if _, err := handler.Handle(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}

	cmd.Title = "Tarde"
	_, err := handler.Handle(context.Background(), cmd)

	if !errors.Is(err, domain.ErrTaskVersionConflict) {
		t.Fatalf("err = %v", err)
	}
	if repo.tasks[id].Title != "Probar" || repo.tasks[id].Version != 2 {
		t.Errorf("tarea = %+v", repo.tasks[id])
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name   string
		id     func(uuid.UUID) string
		body   string
		status int
	}{
		{"válida", uuid.UUID.String, `{"title":"Probar","expected_version":1}`, http.StatusOK},
		{"versión vieja", uuid.UUID.String, `{"title":"Probar","expected_version":7}`, http.StatusConflict},
		{"sin versión", uuid.UUID.String, `{"title":"Probar"}`, http.StatusBadRequest},
		{"versión cero", uuid.UUID.String, `{"title":"Probar","expected_version":0}`, http.StatusBadRequest},
		{"versión negativa", uuid.UUID.String, `{"title":"Probar","expected_version":-1}`, http.StatusBadRequest},
		{"id inválido", func(uuid.UUID) string { return "abc" }, `{"title":"A","expected_version":1}`, http.StatusBadRequest},
		{"no existe", func(uuid.UUID) string { return uuid.NewString() }, `{"title":"A","expected_version":1}`, http.StatusNotFound},
		{"título vacío", uuid.UUID.String, `{"title":"","expected_version":1}`, http.StatusBadRequest},
		{"status no editable", uuid.UUID.String, `{"title":"A","expected_version":1,"status":"ready"}`, http.StatusBadRequest},
		{"blocked no editable", uuid.UUID.String, `{"title":"A","expected_version":1,"blocked":true}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)
			mux := http.NewServeMux()
			New(repo).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/tasks/"+tt.id(id), strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
