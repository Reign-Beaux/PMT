package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

var errNotPlanning = kernel.Conflict("el proyecto no está en planning")

type fakeProjects struct {
	err error
}

func (p fakeProjects) EnsurePlanning(context.Context, uuid.UUID) error {
	return p.err
}

type fakeRepo struct {
	stages map[uuid.UUID]domain.Stage
}

func newRepoWith(t *testing.T) (*fakeRepo, uuid.UUID) {
	t.Helper()
	s, err := domain.NewStage(uuid.New(), "Diseño", "Inicial")
	if err != nil {
		t.Fatal(err)
	}
	return &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}, s.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	s, ok := r.stages[id]
	if !ok {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	return s, nil
}

func (r *fakeRepo) Update(_ context.Context, s domain.Stage) (domain.Stage, error) {
	r.stages[s.ID] = s
	return s, nil
}

func TestHandleUpdatesStage(t *testing.T) {
	repo, id := newRepoWith(t)

	result, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{ID: id, Name: "Desarrollo", Description: "Otra"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Desarrollo" || repo.stages[id].Description != "Otra" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleKeepsStageWhenInvalid(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{ID: id, Name: " "})

	if !errors.Is(err, domain.ErrStageNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if repo.stages[id].Name != "Diseño" {
		t.Errorf("etapa = %+v", repo.stages[id])
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo, fakeProjects{err: errNotPlanning}).Handle(context.Background(), Command{ID: id, Name: "Otra"})

	if !errors.Is(err, errNotPlanning) {
		t.Fatalf("err = %v", err)
	}
	if repo.stages[id].Name != "Diseño" {
		t.Errorf("etapa = %+v", repo.stages[id])
	}
}

func TestRESTRejectsOutsidePlanning(t *testing.T) {
	repo, id := newRepoWith(t)
	mux := http.NewServeMux()
	New(repo, fakeProjects{err: errNotPlanning}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/stages/"+id.String(), strings.NewReader(`{"name":"Otra"}`)))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestREST(t *testing.T) {
	repo, id := newRepoWith(t)
	tests := []struct {
		name   string
		id     string
		body   string
		status int
	}{
		{"válida", id.String(), `{"name":"Desarrollo"}`, http.StatusOK},
		{"id inválido", "abc", `{"name":"A"}`, http.StatusBadRequest},
		{"no existe", uuid.NewString(), `{"name":"A"}`, http.StatusNotFound},
		{"nombre vacío", id.String(), `{"name":""}`, http.StatusBadRequest},
		{"position no editable", id.String(), `{"name":"A","position":2}`, http.StatusBadRequest},
		{"status no editable", id.String(), `{"name":"A","status":"stand_by"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(repo, fakeProjects{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/stages/"+tt.id, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
