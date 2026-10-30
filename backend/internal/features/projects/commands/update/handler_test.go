package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
)

type fakeRepo struct {
	projects map[uuid.UUID]domain.Project
}

func newRepoWith(t *testing.T) (*fakeRepo, uuid.UUID) {
	t.Helper()
	p, err := domain.NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}
	return &fakeRepo{projects: map[uuid.UUID]domain.Project{p.ID: p}}, p.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	p, ok := r.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

func (r *fakeRepo) Update(_ context.Context, p domain.Project) (domain.Project, error) {
	r.projects[p.ID] = p
	return p, nil
}

func TestHandleUpdatesProject(t *testing.T) {
	repo, id := newRepoWith(t)

	result, err := New(repo).Handle(context.Background(), Command{ID: id, Name: "PMT v4", Description: "Otro"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "PMT v4" || repo.projects[id].Description != "Otro" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleKeepsProjectWhenInvalid(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo).Handle(context.Background(), Command{ID: id, Name: " "})

	if !errors.Is(err, domain.ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if repo.projects[id].Name != "PMT" {
		t.Errorf("proyecto = %+v", repo.projects[id])
	}
}

func TestHandleMissingProject(t *testing.T) {
	repo, _ := newRepoWith(t)

	_, err := New(repo).Handle(context.Background(), Command{ID: uuid.New(), Name: "PMT"})

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
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
		{"válido", id.String(), `{"name":"PMT v4"}`, http.StatusOK},
		{"id inválido", "abc", `{"name":"PMT"}`, http.StatusBadRequest},
		{"no existe", uuid.NewString(), `{"name":"PMT"}`, http.StatusNotFound},
		{"nombre vacío", id.String(), `{"name":""}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(repo).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/projects/"+tt.id, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
