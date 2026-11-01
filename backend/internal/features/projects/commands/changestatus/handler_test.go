package changestatus

import (
	"context"
	"encoding/json"
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
	updates  int
	recorded []domain.StatusChange
	reached  bool
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

func (r *fakeRepo) ChangeStatus(_ context.Context, p domain.Project, change domain.StatusChange) (domain.Project, error) {
	r.updates++
	r.recorded = append(r.recorded, change)
	r.projects[p.ID] = p
	return p, nil
}

func (r *fakeRepo) HasReachedWorking(context.Context, uuid.UUID) (bool, error) {
	return r.reached, nil
}

// fakeStages responde como un proyecto con esas etapas y cuenta las consultas.
type fakeStages struct {
	releasable, startable, closed error
	calls                         int
}

func (s *fakeStages) EnsureReleasable(context.Context, uuid.UUID) error {
	s.calls++
	return s.releasable
}

func (s *fakeStages) EnsureStartable(context.Context, uuid.UUID) error {
	s.calls++
	return s.startable
}

func (s *fakeStages) EnsureClosed(context.Context, uuid.UUID) error {
	s.calls++
	return s.closed
}

func TestHandleChangesStatus(t *testing.T) {
	repo, id := newRepoWith(t)

	result, err := New(repo, &fakeStages{}).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: actorFor(domain.StatusStandBy), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusStandBy || repo.projects[id].Status != domain.StatusStandBy {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleRejectedStatusIsNotSaved(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		want   error
	}{
		{"transición no permitida", domain.StatusWorking, domain.ErrProjectStatusTransition},
		{"mismo status", domain.StatusPlanning, domain.ErrProjectStatusTransition},
		{"desconocido", "archived", domain.ErrProjectStatusInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)

			_, err := New(repo, &fakeStages{}).Handle(context.Background(), Command{ID: id, Status: tt.status, Actor: actorFor(tt.status), Reason: "motivo"})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if repo.updates != 0 || repo.projects[id].Status != domain.StatusPlanning {
				t.Errorf("updates = %d, proyecto = %+v", repo.updates, repo.projects[id])
			}
		})
	}
}

func TestHandleMissingProject(t *testing.T) {
	repo, _ := newRepoWith(t)

	_, err := New(repo, &fakeStages{}).Handle(context.Background(), Command{ID: uuid.New(), Status: domain.StatusStandBy, Actor: actorFor(domain.StatusStandBy), Reason: "motivo"})

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name   string
		id     func(uuid.UUID) string
		body   string
		status int
	}{
		{"válido", uuid.UUID.String, `{"status":"stand_by"}`, http.StatusOK},
		{"id inválido", func(uuid.UUID) string { return "abc" }, `{"status":"stand_by"}`, http.StatusBadRequest},
		{"no existe", func(uuid.UUID) string { return uuid.NewString() }, `{"status":"stand_by"}`, http.StatusNotFound},
		{"status desconocido", uuid.UUID.String, `{"status":"archived"}`, http.StatusBadRequest},
		{"sin status", uuid.UUID.String, `{}`, http.StatusBadRequest},
		{"campo desconocido", uuid.UUID.String, `{"status":"stand_by","name":"Otro"}`, http.StatusBadRequest},
		{"transición no permitida", uuid.UUID.String, `{"status":"working"}`, http.StatusConflict},
		{"mismo status", uuid.UUID.String, `{"status":"planning"}`, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)
			mux := http.NewServeMux()
			New(repo, &fakeStages{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/projects/"+tt.id(id)+"/status", strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTReturnsWholeProject(t *testing.T) {
	repo, id := newRepoWith(t)
	p := repo.projects[id]
	p.ProductVersion = "1.0.0"
	repo.projects[id] = p
	mux := http.NewServeMux()
	New(repo, &fakeStages{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/projects/"+id.String()+"/status", strings.NewReader(`{"status":"stand_by"}`)))

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"id", "name", "description", "created_at", "updated_at"} {
		if _, ok := body[field]; !ok {
			t.Errorf("falta %s en %s", field, rec.Body)
		}
	}
	if body["status"] != "stand_by" || body["product_version"] != "1.0.0" {
		t.Errorf("cuerpo = %s", rec.Body)
	}
}
