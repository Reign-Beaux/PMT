package list

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

type fakeReader struct {
	stages []domain.Stage
}

func (r *fakeReader) List(context.Context, uuid.UUID) ([]domain.Stage, error) {
	return r.stages, nil
}

func TestRESTReturnsEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/stages", nil))

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTRejectsInvalidProjectID(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/abc/stages", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestHandleKeepsOrder(t *testing.T) {
	reader := &fakeReader{stages: []domain.Stage{{Name: "A", Position: 1}, {Name: "B", Position: 2}}}

	items, err := New(reader).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Name != "B" || items[1].Position != 2 {
		t.Errorf("items = %+v", items)
	}
}
