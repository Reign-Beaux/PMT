package list

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

type fakeReader struct {
	tasks []domain.Task
}

func (r *fakeReader) List(context.Context, uuid.UUID) ([]domain.Task, error) {
	return r.tasks, nil
}

type fakeStages struct {
	err error
}

func (s fakeStages) Check(context.Context, uuid.UUID) error {
	return s.err
}

func TestRESTReturnsEmptyArray(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeStages{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stages/"+uuid.NewString()+"/tasks", nil))

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTRejectsInvalidStageID(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeStages{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stages/abc/tasks", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestRESTReturnsNotFoundForMissingStage(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}, fakeStages{err: domain.ErrStageNotFound}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stages/"+uuid.NewString()+"/tasks", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestHandleKeepsOrder(t *testing.T) {
	reader := &fakeReader{tasks: []domain.Task{{Title: "A"}, {Title: "B"}}}

	items, err := New(reader, fakeStages{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Title != "B" {
		t.Errorf("items = %+v", items)
	}
}
