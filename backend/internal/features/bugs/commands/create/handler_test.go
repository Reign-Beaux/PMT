package create

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

var errNotWorking = kernel.Conflict("el proyecto no está en working")

type fakeProjects struct {
	err error
}

func (p fakeProjects) EnsureWorking(context.Context, uuid.UUID) error {
	return p.err
}

type fakeRepo struct {
	saved []domain.Bug
}

func (r *fakeRepo) Create(_ context.Context, b domain.Bug) (domain.Bug, error) {
	r.saved = append(r.saved, b)
	return b, nil
}

func TestHandleCreatesBug(t *testing.T) {
	repo := &fakeRepo{}
	projectID := uuid.New()

	result, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{
		ProjectID: projectID, Title: "Falla", StepsToReproduce: "Paso 1", ProductVersion: "1.2.0",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusReported || result.ProjectID != projectID || len(repo.saved) != 1 {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleRejectsOutsideWorking(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeProjects{err: errNotWorking}).Handle(context.Background(), Command{
		ProjectID: uuid.New(), Title: "Falla", StepsToReproduce: "Paso 1", ProductVersion: "1.2.0",
	})

	if !errors.Is(err, errNotWorking) || len(repo.saved) != 0 {
		t.Errorf("err = %v, guardados = %+v", err, repo.saved)
	}
}

func TestHandleRejectsInvalidBug(t *testing.T) {
	_, err := New(&fakeRepo{}, fakeProjects{}).Handle(context.Background(), Command{ProjectID: uuid.New(), Title: "Falla"})

	if !errors.Is(err, domain.ErrBugStepsRequired) {
		t.Errorf("err = %v", err)
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name     string
		projects fakeProjects
		id       string
		body     string
		want     int
	}{
		{"crea", fakeProjects{}, uuid.NewString(), `{"title":"F","steps_to_reproduce":"P","product_version":"1"}`, http.StatusCreated},
		{"fuera de working", fakeProjects{err: errNotWorking}, uuid.NewString(), `{"title":"F","steps_to_reproduce":"P","product_version":"1"}`, http.StatusConflict},
		{"datos inválidos", fakeProjects{}, uuid.NewString(), `{"title":"F"}`, http.StatusBadRequest},
		{"id inválido", fakeProjects{}, "x", `{}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{}, tt.projects).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/projects/"+tt.id+"/bugs", strings.NewReader(tt.body)))

			if rec.Code != tt.want {
				t.Errorf("código = %d: %s", rec.Code, rec.Body)
			}
		})
	}
}
