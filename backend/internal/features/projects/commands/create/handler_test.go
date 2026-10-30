package create

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/features/projects/domain"
)

type fakeRepo struct {
	saved []domain.Project
}

func (r *fakeRepo) Create(_ context.Context, p domain.Project) (domain.Project, error) {
	r.saved = append(r.saved, p)
	return p, nil
}

func TestHandleCreatesProject(t *testing.T) {
	repo := &fakeRepo{}

	result, err := New(repo).Handle(context.Background(), Command{Name: " PMT ", Description: "Gestor"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "PMT" || len(repo.saved) != 1 {
		t.Errorf("result = %+v, guardados = %d", result, len(repo.saved))
	}
}

func TestHandleRejectsInvalidProject(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo).Handle(context.Background(), Command{Name: " "})

	if !errors.Is(err, domain.ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar el proyecto")
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"válido", `{"name":"PMT"}`, http.StatusCreated},
		{"nombre vacío", `{"name":" "}`, http.StatusBadRequest},
		{"json inválido", `{`, http.StatusBadRequest},
		{"campo desconocido", `{"name":"A","x":1}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
