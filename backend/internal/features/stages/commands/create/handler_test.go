package create

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

func (c fakeProjects) EnsurePlanning(context.Context, uuid.UUID) error {
	return c.err
}

type fakeRepo struct {
	saved []domain.Stage
	err   error
}

func (r *fakeRepo) Create(_ context.Context, s domain.Stage) (domain.Stage, error) {
	if r.err != nil {
		return domain.Stage{}, r.err
	}
	r.saved = append(r.saved, s)
	return s, nil
}

func TestHandleCreatesStage(t *testing.T) {
	repo := &fakeRepo{}
	projectID := uuid.New()

	result, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{Actor: kernel.Owner, ProjectID: projectID, Name: " Diseño "})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Diseño" || result.ProjectID != projectID || len(repo.saved) != 1 {
		t.Errorf("result = %+v, guardadas = %d", result, len(repo.saved))
	}
	if result.Status != domain.StatusPlanning {
		t.Errorf("status = %s", result.Status)
	}
}

func TestHandleRejectsInvalidStage(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{Actor: kernel.Owner, ProjectID: uuid.New(), Name: " "})

	if !errors.Is(err, domain.ErrStageNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar la etapa")
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeProjects{err: errNotPlanning}).Handle(context.Background(), Command{Actor: kernel.Owner, ProjectID: uuid.New(), Name: "Diseño"})

	if !errors.Is(err, errNotPlanning) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar")
	}
}

func TestRESTRejectsOutsidePlanning(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeRepo{}, fakeProjects{err: errNotPlanning}).Register(mux)
	rec := httptest.NewRecorder()
	path := "/api/projects/" + uuid.NewString() + "/stages"

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"A"}`)))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestREST(t *testing.T) {
	path := "/api/projects/" + uuid.NewString() + "/stages"
	tests := []struct {
		name   string
		path   string
		body   string
		err    error
		status int
	}{
		{"válida", path, `{"name":"Diseño"}`, nil, http.StatusCreated},
		{"nombre vacío", path, `{"name":" "}`, nil, http.StatusBadRequest},
		{"json inválido", path, `{`, nil, http.StatusBadRequest},
		{"proyecto inválido", "/api/projects/abc/stages", `{"name":"A"}`, nil, http.StatusBadRequest},
		{"proyecto inexistente", path, `{"name":"A"}`, domain.ErrProjectNotFound, http.StatusNotFound},
		{"posición ocupada", path, `{"name":"A"}`, domain.ErrPositionTaken, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{err: tt.err}, fakeProjects{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
