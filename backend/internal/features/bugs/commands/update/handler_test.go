package update

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

var planner = kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}

type fakeProjects struct {
	err error
}

func (p fakeProjects) EnsureWorking(context.Context, uuid.UUID) error {
	return p.err
}

type fakeRepo struct {
	bug domain.Bug
}

func newRepo(t *testing.T) *fakeRepo {
	t.Helper()
	b, err := domain.NewBug(uuid.New(), "Falla", "", "Paso 1", "1.2.0", "pp")
	if err != nil {
		t.Fatal(err)
	}
	return &fakeRepo{bug: b}
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Bug, error) {
	if id != r.bug.ID {
		return domain.Bug{}, domain.ErrBugNotFound
	}
	return r.bug, nil
}

func (r *fakeRepo) Update(_ context.Context, b domain.Bug) (domain.Bug, error) {
	r.bug = b
	return b, nil
}

func TestHandleUpdatesBug(t *testing.T) {
	repo := newRepo(t)
	repo.bug.Status = domain.StatusConfirmed

	result, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{
		ID: repo.bug.ID, Title: "Otra", StepsToReproduce: "Paso 2", ProductVersion: "1.3.0", Actor: planner,
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Otra" || result.ProductVersion != "1.3.0" || result.Status != domain.StatusConfirmed {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleRejects(t *testing.T) {
	repo := newRepo(t)
	valid := Command{ID: repo.bug.ID, Title: "Otra", StepsToReproduce: "P", ProductVersion: "1", Actor: planner}
	missing := valid
	missing.ID = uuid.New()
	invalid := valid
	invalid.ProductVersion = ""

	tests := []struct {
		name     string
		projects fakeProjects
		cmd      Command
		want     error
	}{
		{"fuera de working", fakeProjects{err: errNotWorking}, valid, errNotWorking},
		{"no existe", fakeProjects{}, missing, domain.ErrBugNotFound},
		{"datos inválidos", fakeProjects{}, invalid, domain.ErrBugVersionRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(repo, tt.projects).Handle(context.Background(), tt.cmd); !errors.Is(err, tt.want) {
				t.Errorf("err = %v", err)
			}
			if repo.bug.Title != "Falla" {
				t.Errorf("bug = %+v", repo.bug)
			}
		})
	}
}

func TestHandleRejectsClosedBug(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusDone, domain.StatusRejected} {
		repo := newRepo(t)
		repo.bug.Status = status

		_, err := New(repo, fakeProjects{}).Handle(context.Background(), Command{
			ID: repo.bug.ID, Title: "Otra", StepsToReproduce: "P", ProductVersion: "1", Actor: planner,
		})

		if !errors.Is(err, domain.ErrBugClosed) || repo.bug.Title != "Falla" {
			t.Errorf("%s: err = %v, bug = %+v", status, err, repo.bug)
		}
	}
}

func TestHandleChecksProfile(t *testing.T) {
	allowed := []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser}
	rejected := []kernel.Profile{
		kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDevFrontend, kernel.ProfileDevBackend,
		kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA, "",
	}
	command := func(id uuid.UUID, profile kernel.Profile) Command {
		return Command{ID: id, Title: "Otra", StepsToReproduce: "P", ProductVersion: "1", Actor: kernel.Actor{Name: "agente", Profile: profile}}
	}

	for _, profile := range allowed {
		repo := newRepo(t)
		if _, err := New(repo, fakeProjects{}).Handle(context.Background(), command(repo.bug.ID, profile)); err != nil || repo.bug.Title != "Otra" {
			t.Errorf("%s: err = %v, bug = %+v", profile, err, repo.bug)
		}
	}
	for _, profile := range rejected {
		repo := newRepo(t)
		// Un ID inexistente confirma que el perfil se comprueba primero.
		_, err := New(repo, fakeProjects{}).Handle(context.Background(), command(uuid.New(), profile))
		if !errors.Is(err, domain.ErrUpdateForbidden) {
			t.Errorf("%s: err = %v", profile, err)
		}
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name    string
		profile kernel.Profile
		status  domain.Status
		want    int
	}{
		{"corrige", kernel.ProfilePP, domain.StatusReported, http.StatusOK},
		{"perfil rechazado", kernel.ProfileQA, domain.StatusReported, http.StatusForbidden},
		{"bug cerrado", kernel.ProfileUser, domain.StatusDone, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo(t)
			repo.bug.Status = tt.status
			mux := http.NewServeMux()
			New(repo, fakeProjects{}).Register(mux)
			rec := httptest.NewRecorder()

			body := `{"title":"F","steps_to_reproduce":"P","product_version":"1"}`
			req := httptest.NewRequest(http.MethodPut, "/api/bugs/"+repo.bug.ID.String(), strings.NewReader(body))
			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), kernel.Actor{Name: "a", Profile: tt.profile})))

			if rec.Code != tt.want {
				t.Errorf("código = %d: %s", rec.Code, rec.Body)
			}
		})
	}
}
