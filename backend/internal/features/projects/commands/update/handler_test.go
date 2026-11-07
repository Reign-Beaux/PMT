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
	"pmt/internal/kernel"
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

	result, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Name: "PMT v4", Description: "Otro"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "PMT v4" || repo.projects[id].Description != "Otro" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleReplacesProductVersion(t *testing.T) {
	repo, id := newRepoWith(t)

	result, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Name: "PMT", ProductVersion: " 1.0.0 "})

	if err != nil {
		t.Fatal(err)
	}
	if result.ProductVersion != "1.0.0" {
		t.Errorf("result = %+v", result)
	}

	result, err = New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Name: "PMT"})

	if err != nil {
		t.Fatal(err)
	}
	if result.ProductVersion != "" || repo.projects[id].ProductVersion != "" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleKeepsProjectWhenInvalid(t *testing.T) {
	repo, id := newRepoWith(t)

	_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Name: " "})

	if !errors.Is(err, domain.ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if repo.projects[id].Name != "PMT" {
		t.Errorf("proyecto = %+v", repo.projects[id])
	}
}

func TestHandleMissingProject(t *testing.T) {
	repo, _ := newRepoWith(t)

	_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: uuid.New(), Name: "PMT"})

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleOutsidePlanning(t *testing.T) {
	statuses := []domain.Status{
		domain.StatusStandBy, domain.StatusInDev, domain.StatusWorking, domain.StatusStopped, domain.StatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			repo, id := newRepoWith(t)
			p := repo.projects[id]
			p.Status = status
			p.ProductVersion = "1.0.0"
			repo.projects[id] = p

			result, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Name: "Otro", Description: "Nueva", ProductVersion: "1.0.0"})

			if err != nil {
				t.Fatal(err)
			}
			if result.Name != "Otro" || result.Description != "Nueva" {
				t.Errorf("result = %+v", result)
			}

			_, err = New(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Name: "Tercero", ProductVersion: "2.0.0"})

			if !errors.Is(err, domain.ErrProductVersionFrozen) {
				t.Fatalf("err = %v", err)
			}
			if repo.projects[id].Name != "Otro" || repo.projects[id].ProductVersion != "1.0.0" {
				t.Errorf("proyecto = %+v", repo.projects[id])
			}
		})
	}
}

func TestRESTOutsidePlanning(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"nombre", `{"name":"Otro"}`, http.StatusOK},
		{"versión de producto", `{"name":"Otro","product_version":"2.0.0"}`, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)
			p := repo.projects[id]
			p.Status = domain.StatusStandBy
			repo.projects[id] = p
			mux := http.NewServeMux()
			New(repo).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/projects/"+id.String(), strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
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
		{"con versión de producto", id.String(), `{"name":"PMT","product_version":"1.0.0"}`, http.StatusOK},
		{"status no editable", id.String(), `{"name":"PMT","status":"stopped"}`, http.StatusBadRequest},
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
