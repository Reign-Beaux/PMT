package list

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	projectsdomain "pmt/internal/features/projects/domain"
	"pmt/internal/features/stages/domain"
)

type fakeReader struct {
	stages []domain.Stage
	called bool
}

func (r *fakeReader) List(context.Context, uuid.UUID) ([]domain.Stage, error) {
	r.called = true
	return r.stages, nil
}

type fakeProjects struct {
	err error
}

func (p fakeProjects) Check(context.Context, uuid.UUID) error {
	return p.err
}

func TestRESTReturnsEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeProjects{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/stages", nil))

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTRejectsInvalidProjectID(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeProjects{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/abc/stages", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTReturnsNotFoundForMissingProject(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeProjects{err: projectsdomain.ErrProjectNotFound}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/stages", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestHandleReturnsNotFoundWithoutListing(t *testing.T) {
	reader := &fakeReader{stages: []domain.Stage{{Name: "A"}}}

	_, err := New(reader, fakeProjects{err: projectsdomain.ErrProjectNotFound}).Handle(context.Background(), uuid.New())

	if !errors.Is(err, projectsdomain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
	}
	if reader.called {
		t.Error("reader.List llamado con proyecto inexistente")
	}
}

func TestHandleKeepsOrder(t *testing.T) {
	reader := &fakeReader{stages: []domain.Stage{{Name: "A", Position: 1}, {Name: "B", Position: 2}}}

	items, err := New(reader, fakeProjects{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Name != "B" || items[1].Position != 2 {
		t.Errorf("items = %+v", items)
	}
}

func TestHandleReturnsEmptyForProjectWithoutStages(t *testing.T) {
	items, err := New(&fakeReader{}, fakeProjects{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("items = %+v", items)
	}
}
