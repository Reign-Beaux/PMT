package liststatuschanges

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

type fakeReader struct {
	changes []domain.StatusChange
	err     error
}

func (r fakeReader) ListStatusChanges(context.Context, uuid.UUID) ([]domain.StatusChange, error) {
	return r.changes, r.err
}

func TestHandleMapsChanges(t *testing.T) {
	id := uuid.New()
	change := domain.StatusChange{
		ID: uuid.New(), TaskID: id, From: "a", To: "b",
		ActorName: "agente", ActorProfile: "PP", Reason: "motivo", ChangedAt: time.Now(),
	}

	items, err := New(fakeReader{changes: []domain.StatusChange{change}}).Handle(context.Background(), id)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	got := items[0]
	if got.TaskID != id || got.FromStatus != "a" || got.ToStatus != "b" || got.ActorName != "agente" || got.ActorProfile != "PP" || got.Reason != "motivo" {
		t.Errorf("item = %+v", got)
	}
}

func TestHandleEmptyIsNotNil(t *testing.T) {
	items, err := New(fakeReader{}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Errorf("items = %#v", items)
	}
}

func TestHandlePropagatesError(t *testing.T) {
	want := errors.New("fallo")

	if _, err := New(fakeReader{err: want}).Handle(context.Background(), uuid.New()); !errors.Is(err, want) {
		t.Errorf("error = %v", err)
	}
}

func TestRESTListsChanges(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeReader{}).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/tasks/"+uuid.NewString()+"/status-changes", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Errorf("cuerpo = %q, %v", rec.Body.String(), err)
	}
}

func TestRESTRejectsBadID(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeReader{}).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/tasks/x/status-changes", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}
