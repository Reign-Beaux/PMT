package get

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
)

type fakeReader struct {
	task domain.Task
}

func (r *fakeReader) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	if id != r.task.ID {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return r.task, nil
}

func TestREST(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name   string
		id     string
		status int
	}{
		{"existe", id.String(), http.StatusOK},
		{"id inválido", "abc", http.StatusBadRequest},
		{"no existe", uuid.NewString(), http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{task: domain.Task{ID: id}}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/tasks/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTIncludesDoneAtOnlyWhenDone(t *testing.T) {
	when := time.Date(2026, 11, 8, 12, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		task       domain.Task
		wantInBody string
		wantKey    bool
	}{
		{
			name:       "done trae done_at en RFC 3339",
			task:       domain.Task{ID: uuid.New(), Status: domain.StatusDone, DoneAt: &when},
			wantInBody: `"done_at":"2026-11-08T12:30:00Z"`,
			wantKey:    true,
		},
		{
			name:    "otro status omite la clave",
			task:    domain.Task{ID: uuid.New(), Status: domain.StatusInDev},
			wantKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{task: tt.task}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/tasks/"+tt.task.ID.String(), nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
			if tt.wantKey && !strings.Contains(rec.Body.String(), tt.wantInBody) {
				t.Errorf("cuerpo sin %s: %s", tt.wantInBody, rec.Body)
			}
			if !tt.wantKey && strings.Contains(rec.Body.String(), "done_at") {
				t.Errorf("el cuerpo no debe traer done_at: %s", rec.Body)
			}
		})
	}
}
