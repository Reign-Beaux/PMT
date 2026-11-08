package changeproductversion

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
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

type fakeRepo struct {
	projects map[uuid.UUID]domain.Project
	reads    int
	recorded []domain.ProductVersionChange
	saveErr  error
}

func newRepoWith(t *testing.T, status domain.Status, version string) (*fakeRepo, uuid.UUID) {
	t.Helper()
	p, err := domain.NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}
	p.Status = status
	p.ProductVersion = version
	return &fakeRepo{projects: map[uuid.UUID]domain.Project{p.ID: p}}, p.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	r.reads++
	p, ok := r.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

// ChangeProductVersion guarda versión y registro juntos, o nada, como la transacción real.
func (r *fakeRepo) ChangeProductVersion(_ context.Context, change domain.ProductVersionChange) (domain.Project, error) {
	if r.saveErr != nil {
		return domain.Project{}, r.saveErr
	}
	p := r.projects[change.ProjectID]
	p.ProductVersion = change.To
	r.projects[p.ID] = p
	r.recorded = append(r.recorded, change)
	return p, nil
}

var pp = kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}

func TestHandleChangesVersionAndRecords(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusWorking, "1.0.0")

	result, err := New(repo).Handle(context.Background(), Command{ID: id, ProductVersion: "1.1.0", Reason: " etapa nueva ", Actor: pp})

	if err != nil {
		t.Fatal(err)
	}
	if result.ProductVersion != "1.1.0" || repo.projects[id].ProductVersion != "1.1.0" {
		t.Errorf("result = %+v", result)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("registros = %d", len(repo.recorded))
	}
	got := repo.recorded[0]
	if got.ProjectID != id || got.From != "1.0.0" || got.To != "1.1.0" || got.ActorName != "pp" || got.ActorProfile != "PP" || got.Reason != "etapa nueva" {
		t.Errorf("registro = %+v", got)
	}
}

func TestHandleUserCanChange(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusWorking, "")

	if _, err := New(repo).Handle(context.Background(), Command{ID: id, ProductVersion: "0.1.0", Reason: "inicio", Actor: kernel.Owner}); err != nil {
		t.Fatal(err)
	}
}

func TestHandleRejectsOtherProfilesBeforeReading(t *testing.T) {
	for _, profile := range []kernel.Profile{kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDP, kernel.ProfileDevFrontend, kernel.ProfileDevBackend, kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA} {
		t.Run(string(profile), func(t *testing.T) {
			repo, id := newRepoWith(t, domain.StatusWorking, "1.0.0")

			_, err := New(repo).Handle(context.Background(), Command{ID: id, ProductVersion: "1.1.0", Reason: "x", Actor: kernel.Actor{Name: "a", Profile: profile}})

			if !errors.Is(err, domain.ErrPlanForbidden) {
				t.Fatalf("err = %v", err)
			}
			if repo.reads != 0 || len(repo.recorded) != 0 {
				t.Errorf("lecturas = %d, registros = %d", repo.reads, len(repo.recorded))
			}
		})
	}
}

func TestHandleRejectedIsNotSaved(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.Status
		version string
		reason  string
		want    error
	}{
		{"planning", domain.StatusPlanning, "1.1.0", "x", domain.ErrProductVersionNotWorking},
		{"in_dev", domain.StatusInDev, "1.1.0", "x", domain.ErrProductVersionNotWorking},
		{"stopped", domain.StatusStopped, "1.1.0", "x", domain.ErrProductVersionNotWorking},
		{"formato", domain.StatusWorking, "1.1", "x", domain.ErrProductVersionFormat},
		{"igual", domain.StatusWorking, "1.0.0", "x", domain.ErrProductVersionNotGreater},
		{"menor", domain.StatusWorking, "0.9.9", "x", domain.ErrProductVersionNotGreater},
		{"motivo vacío", domain.StatusWorking, "1.1.0", " ", domain.ErrProductVersionReasonRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.status, "1.0.0")

			_, err := New(repo).Handle(context.Background(), Command{ID: id, ProductVersion: tt.version, Reason: tt.reason, Actor: pp})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.recorded) != 0 || repo.projects[id].ProductVersion != "1.0.0" {
				t.Errorf("registros = %d, proyecto = %+v", len(repo.recorded), repo.projects[id])
			}
		})
	}
}

func TestHandleSaveFailureLeavesNothing(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusWorking, "1.0.0")
	repo.saveErr = domain.ErrProductVersionChanged

	_, err := New(repo).Handle(context.Background(), Command{ID: id, ProductVersion: "1.1.0", Reason: "x", Actor: pp})

	if !errors.Is(err, domain.ErrProductVersionChanged) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.recorded) != 0 || repo.projects[id].ProductVersion != "1.0.0" {
		t.Errorf("registros = %d, proyecto = %+v", len(repo.recorded), repo.projects[id])
	}
}

func TestHandleMissingProject(t *testing.T) {
	repo, _ := newRepoWith(t, domain.StatusWorking, "1.0.0")

	_, err := New(repo).Handle(context.Background(), Command{ID: uuid.New(), ProductVersion: "1.1.0", Reason: "x", Actor: pp})

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		id     func(uuid.UUID) string
		body   string
		code   int
	}{
		{"válido", domain.StatusWorking, uuid.UUID.String, `{"product_version":"1.1.0","reason":"etapa"}`, http.StatusOK},
		{"id inválido", domain.StatusWorking, func(uuid.UUID) string { return "abc" }, `{"product_version":"1.1.0","reason":"etapa"}`, http.StatusBadRequest},
		{"no existe", domain.StatusWorking, func(uuid.UUID) string { return uuid.NewString() }, `{"product_version":"1.1.0","reason":"etapa"}`, http.StatusNotFound},
		{"fuera de working", domain.StatusPlanning, uuid.UUID.String, `{"product_version":"1.1.0","reason":"etapa"}`, http.StatusConflict},
		{"formato", domain.StatusWorking, uuid.UUID.String, `{"product_version":"v1.1.0","reason":"etapa"}`, http.StatusBadRequest},
		{"no mayor", domain.StatusWorking, uuid.UUID.String, `{"product_version":"1.0.0","reason":"etapa"}`, http.StatusBadRequest},
		{"sin motivo", domain.StatusWorking, uuid.UUID.String, `{"product_version":"1.1.0"}`, http.StatusBadRequest},
		{"campo desconocido", domain.StatusWorking, uuid.UUID.String, `{"product_version":"1.1.0","reason":"etapa","name":"x"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.status, "1.0.0")
			mux := http.NewServeMux()
			New(repo).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/projects/"+tt.id(id)+"/product-version", strings.NewReader(tt.body)))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTReturnsWholeProject(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusWorking, "1.0.0")
	mux := http.NewServeMux()
	New(repo).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/projects/"+id.String()+"/product-version", strings.NewReader(`{"product_version":"2.0.0","reason":"cambio grande"}`)))

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"id", "name", "description", "created_at", "updated_at"} {
		if _, ok := body[field]; !ok {
			t.Errorf("falta %s en %s", field, rec.Body)
		}
	}
	if body["status"] != "working" || body["product_version"] != "2.0.0" {
		t.Errorf("cuerpo = %s", rec.Body)
	}
}

func TestRESTRejectsOtherProfile(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusWorking, "1.0.0")
	mux := http.NewServeMux()
	New(repo).Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+id.String()+"/product-version", strings.NewReader(`{"product_version":"1.1.0","reason":"x"}`))
	req = req.WithContext(rest.WithActor(req.Context(), kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || repo.reads != 0 {
		t.Errorf("status = %d, lecturas = %d, cuerpo = %s", rec.Code, repo.reads, rec.Body)
	}
}
