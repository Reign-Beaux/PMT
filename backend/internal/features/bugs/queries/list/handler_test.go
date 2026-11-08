package list

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
	projectsdomain "pmt/internal/features/projects/domain"
)

type fakeReader struct {
	bugs   []domain.Bug
	called bool
}

func (r *fakeReader) List(context.Context, uuid.UUID) ([]domain.Bug, error) {
	r.called = true
	return r.bugs, nil
}

type fakeProjects struct {
	err error
}

func (p fakeProjects) Check(context.Context, uuid.UUID) error {
	return p.err
}

func TestHandleListsBugs(t *testing.T) {
	items, err := New(&fakeReader{bugs: []domain.Bug{{Title: "A"}, {Title: "B"}}}, fakeProjects{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Title != "B" {
		t.Errorf("items = %+v", items)
	}
}

func TestHandleReturnsNotFoundWithoutListing(t *testing.T) {
	reader := &fakeReader{bugs: []domain.Bug{{Title: "A"}}}

	_, err := New(reader, fakeProjects{err: projectsdomain.ErrProjectNotFound}).Handle(context.Background(), uuid.New())

	if !errors.Is(err, projectsdomain.ErrProjectNotFound) {
		t.Fatalf("err = %v", err)
	}
	if reader.called {
		t.Error("reader.List llamado con proyecto inexistente")
	}
}

func TestHandleReturnsEmptyForProjectWithoutBugs(t *testing.T) {
	items, err := New(&fakeReader{}, fakeProjects{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("items = %+v", items)
	}
}

func TestRESTReturnsEmptyList(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeProjects{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/bugs", nil))

	var items []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || items == nil || len(items) != 0 {
		t.Errorf("código = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTReturnsNotFoundForMissingProject(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeProjects{err: projectsdomain.ErrProjectNotFound}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/bugs", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("código = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTRejectsInvalidProjectID(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeProjects{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/abc/bugs", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("código = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
