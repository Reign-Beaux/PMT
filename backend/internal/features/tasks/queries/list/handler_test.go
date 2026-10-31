package list

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

type fakeReader struct {
	tasks  []domain.Task
	status domain.Status
}

func (r *fakeReader) ListByStatus(_ context.Context, _ uuid.UUID, status domain.Status) ([]domain.Task, error) {
	r.status = status
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

	items, err := New(reader, fakeStages{}).Handle(context.Background(), uuid.New(), "")

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Title != "B" {
		t.Errorf("items = %+v", items)
	}
}

func TestHandleFiltersByStatus(t *testing.T) {
	reader := &fakeReader{}

	if _, err := New(reader, fakeStages{}).Handle(context.Background(), uuid.New(), domain.StatusInReview); err != nil {
		t.Fatal(err)
	}
	if reader.status != domain.StatusInReview {
		t.Errorf("status = %s", reader.status)
	}
}

func TestHandleRejectsUnknownStatus(t *testing.T) {
	_, err := New(&fakeReader{}, fakeStages{}).Handle(context.Background(), uuid.New(), "archived")

	if !errors.Is(err, domain.ErrTaskStatusInvalid) {
		t.Fatalf("err = %v", err)
	}
}
