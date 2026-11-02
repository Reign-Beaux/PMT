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
	"pmt/internal/platform/rest"
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

var planner = kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}

func TestHandleCreatesBug(t *testing.T) {
	repo := &fakeRepo{}
	projectID := uuid.New()

	result, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{
		ProjectID: projectID, Title: "Falla", StepsToReproduce: "Paso 1", ProductVersion: "1.2.0", Actor: planner,
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusReported || result.ProjectID != projectID || result.ReportedBy != planner.Name || len(repo.saved) != 1 {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleRejectsOutsideWorking(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeProjects{err: errNotWorking}).Handle(context.Background(), Command{
		ProjectID: uuid.New(), Title: "Falla", StepsToReproduce: "Paso 1", ProductVersion: "1.2.0", Actor: planner,
	})

	if !errors.Is(err, errNotWorking) || len(repo.saved) != 0 {
		t.Errorf("err = %v, guardados = %+v", err, repo.saved)
	}
}

func TestHandleRejectsInvalidBug(t *testing.T) {
	_, err := New(&fakeRepo{}, fakeProjects{}).Handle(context.Background(), Command{ProjectID: uuid.New(), Title: "Falla", Actor: planner})

	if !errors.Is(err, domain.ErrBugStepsRequired) {
		t.Errorf("err = %v", err)
	}
}

func TestHandleChecksProfile(t *testing.T) {
	allowed := []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
	rejected := []kernel.Profile{
		kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDevFrontend, kernel.ProfileDevBackend,
		kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA, "",
	}
	command := func(profile kernel.Profile) Command {
		return Command{
			ProjectID: uuid.New(), Title: "Falla", StepsToReproduce: "Paso 1", ProductVersion: "1.2.0",
			Actor: kernel.Actor{Name: "agente", Profile: profile},
		}
	}

	for _, profile := range allowed {
		repo := &fakeRepo{}
		if _, err := New(repo, fakeProjects{}).Handle(context.Background(), command(profile)); err != nil || len(repo.saved) != 1 {
			t.Errorf("%s: err = %v, guardados = %d", profile, err, len(repo.saved))
		}
	}
	for _, profile := range rejected {
		repo := &fakeRepo{}
		// El proyecto fuera de working confirma que el perfil se comprueba primero.
		_, err := New(repo, fakeProjects{err: errNotWorking}).Handle(context.Background(), command(profile))
		if !errors.Is(err, domain.ErrCreateForbidden) || len(repo.saved) != 0 {
			t.Errorf("%s: err = %v, guardados = %d", profile, err, len(repo.saved))
		}
	}
}

func TestRESTRejectsProfile(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeRepo{}, fakeProjects{}).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+uuid.NewString()+"/bugs",
		strings.NewReader(`{"title":"F","steps_to_reproduce":"P","product_version":"1"}`))
	req = req.WithContext(rest.WithActor(req.Context(), kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("código = %d: %s", rec.Code, rec.Body)
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
