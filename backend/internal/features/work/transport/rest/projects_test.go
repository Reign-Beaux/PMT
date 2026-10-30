package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/work/domain"
	"pmt/internal/features/work/usecase"
)

type fakeRepo struct {
	projects map[uuid.UUID]domain.Project
}

func (r *fakeRepo) Create(_ context.Context, p domain.Project) (domain.Project, error) {
	r.projects[p.ID] = p
	return p, nil
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	p, ok := r.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

func (r *fakeRepo) List(context.Context) ([]domain.Project, error) {
	return []domain.Project{}, nil
}

func (r *fakeRepo) Update(_ context.Context, p domain.Project) (domain.Project, error) {
	r.projects[p.ID] = p
	return p, nil
}

func (r *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.projects[id]; !ok {
		return domain.ErrProjectNotFound
	}
	delete(r.projects, id)
	return nil
}

func newTestMux() *http.ServeMux {
	repo := &fakeRepo{projects: map[uuid.UUID]domain.Project{}}
	mux := http.NewServeMux()
	NewProjects(usecase.NewProjects(repo)).Register(mux)
	return mux
}

func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestProjectLifecycle(t *testing.T) {
	mux := newTestMux()

	rec := do(mux, http.MethodPost, "/api/projects", `{"name":"PMT","description":"Gestor"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("crear: status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
	var created projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	path := "/api/projects/" + created.ID.String()

	if rec := do(mux, http.MethodPut, path, `{"name":"PMT v4"}`); rec.Code != http.StatusOK {
		t.Fatalf("actualizar: status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
	if rec := do(mux, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("eliminar: status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
	if rec := do(mux, http.MethodGet, path, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("obtener: status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestProjectErrors(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{"nombre vacío", http.MethodPost, "/api/projects", `{"name":" "}`, http.StatusBadRequest},
		{"json inválido", http.MethodPost, "/api/projects", `{`, http.StatusBadRequest},
		{"campo desconocido", http.MethodPost, "/api/projects", `{"name":"A","x":1}`, http.StatusBadRequest},
		{"id inválido", http.MethodGet, "/api/projects/abc", "", http.StatusBadRequest},
		{"no existe", http.MethodDelete, "/api/projects/" + uuid.NewString(), "", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestMux(), tt.method, tt.path, tt.body)
			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
