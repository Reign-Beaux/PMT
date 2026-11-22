package create

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

// saulID es el usuario con el que actúan las pruebas.
var saulID = uuid.MustParse("00000000-0000-0000-0000-00000000000a")

type fakeRepo struct {
	saved  []domain.Project
	owners []uuid.UUID
}

func (r *fakeRepo) Create(_ context.Context, p domain.Project, owner uuid.UUID) (domain.Project, error) {
	r.saved = append(r.saved, p)
	r.owners = append(r.owners, owner)
	return p, nil
}

func TestHandleLeavesCreatorAsOwner(t *testing.T) {
	repo := &fakeRepo{}

	if _, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Actor{Name: "saul", Profile: kernel.ProfilePP, UserID: saulID}, Name: "PMT"}); err != nil {
		t.Fatal(err)
	}
	if len(repo.owners) != 1 || repo.owners[0] != saulID {
		t.Errorf("owners = %v", repo.owners)
	}
}

func TestHandleRejectsActorWithoutUser(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Actor{Name: "dbp-pp", Profile: kernel.ProfilePP}, Name: "PMT"})

	if !errors.Is(err, domain.ErrNoUser) || len(repo.saved) != 0 {
		t.Fatalf("err = %v, guardados = %d", err, len(repo.saved))
	}
}

func TestHandleCreatesProject(t *testing.T) {
	repo := &fakeRepo{}

	result, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Actor{Name: "saul", Profile: kernel.ProfileUser, UserID: saulID}, Name: " PMT ", Description: "Gestor"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "PMT" || len(repo.saved) != 1 {
		t.Errorf("result = %+v, guardados = %d", result, len(repo.saved))
	}
}

func TestHandleStartsInPlanningWithoutProductVersion(t *testing.T) {
	result, err := New(&fakeRepo{}).Handle(context.Background(), Command{Actor: kernel.Actor{Name: "saul", Profile: kernel.ProfileUser, UserID: saulID}, Name: "PMT"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusPlanning || result.ProductVersion != "" {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleRejectsInvalidProject(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo).Handle(context.Background(), Command{Actor: kernel.Actor{Name: "saul", Profile: kernel.ProfileUser, UserID: saulID}, Name: " "})

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

			mux.ServeHTTP(rec, asUser(httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(tt.body))))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
