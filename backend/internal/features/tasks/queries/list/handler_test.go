package list

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestRESTIncludesDoneAtOnlyForDone(t *testing.T) {
	when := time.Date(2026, 11, 8, 12, 30, 0, 0, time.UTC)
	reader := &fakeReader{tasks: []domain.Task{
		{Title: "cerrada", Status: domain.StatusDone, DoneAt: &when},
		{Title: "abierta", Status: domain.StatusInDev},
	}}
	mux := http.NewServeMux()
	New(reader, fakeStages{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stages/"+uuid.NewString()+"/tasks", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
	var items []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].DoneAt == nil || !items[0].DoneAt.Equal(when) {
		t.Errorf("done: DoneAt = %v, quiero %v", items[0].DoneAt, when)
	}
	if items[1].DoneAt != nil {
		t.Errorf("in_dev: DoneAt = %v, quiero nil", items[1].DoneAt)
	}
	if !strings.Contains(rec.Body.String(), `"done_at":"2026-11-08T12:30:00Z"`) {
		t.Errorf("el JSON no trae done_at en RFC 3339: %s", rec.Body)
	}
	if got := strings.Count(rec.Body.String(), "done_at"); got != 1 {
		t.Errorf("la clave done_at aparece %d veces, quiero 1: %s", got, rec.Body)
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
