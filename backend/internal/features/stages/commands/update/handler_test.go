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
)

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

	result, err := New(repo).Handle(context.Background(), Command{ID: id, Name: "Desarrollo", Description: "Otra"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Desarrollo" || repo.stages[id].Description != "Otra" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleKeepsStageWhenInvalid(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo).Handle(context.Background(), Command{ID: id, Name: " "})

	if !errors.Is(err, domain.ErrStageNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if repo.stages[id].Name != "Diseño" {
		t.Errorf("etapa = %+v", repo.stages[id])
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(repo).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/stages/"+tt.id, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
