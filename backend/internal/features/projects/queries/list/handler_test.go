package list

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/features/projects/domain"
)

type fakeReader struct {
	projects []domain.Project
}

func (r *fakeReader) List(context.Context) ([]domain.Project, error) {
	return r.projects, nil
}

func TestRESTReturnsEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestHandleReturnsProjects(t *testing.T) {
	reader := &fakeReader{projects: []domain.Project{{Name: "A"}, {Name: "B", Status: domain.StatusInDev, ProductVersion: "1.0.0"}}}

	items, err := New(reader).Handle(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Name != "B" || items[1].Status != domain.StatusInDev || items[1].ProductVersion != "1.0.0" {
		t.Errorf("items = %+v", items)
	}
}
