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
)

var errNotWorking = kernel.Conflict("el proyecto no está en working")

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
	b, err := domain.NewBug(uuid.New(), "Falla", "", "Paso 1", "1.2.0")
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
		ID: repo.bug.ID, Title: "Otra", StepsToReproduce: "Paso 2", ProductVersion: "1.3.0",
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
	valid := Command{ID: repo.bug.ID, Title: "Otra", StepsToReproduce: "P", ProductVersion: "1"}
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

func TestREST(t *testing.T) {
	repo := newRepo(t)
	mux := http.NewServeMux()
	New(repo, fakeProjects{}).Register(mux)
	rec := httptest.NewRecorder()

	body := `{"title":"F","steps_to_reproduce":"P","product_version":"1"}`
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/bugs/"+repo.bug.ID.String(), strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Errorf("código = %d: %s", rec.Code, rec.Body)
	}
}
