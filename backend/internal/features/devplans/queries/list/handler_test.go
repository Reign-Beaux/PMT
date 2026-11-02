package list

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/devplans/domain"
)

type fakeReader struct {
	plans []domain.Plan
	calls int
}

func (f *fakeReader) List(context.Context, *uuid.UUID, *uuid.UUID, *uuid.UUID) ([]domain.Plan, error) {
	f.calls++
	return f.plans, nil
}

func TestHandle(t *testing.T) {
	id := uuid.New()
	plan := domain.Plan{ID: uuid.New(), Latest: domain.Version{Version: 2, Content: domain.Content{Scope: "S"}}}
	tests := []struct {
		name  string
		query Query
		want  error
		calls int
	}{
		{"por etapa", Query{StageID: &id}, nil, 1},
		{"por tarea", Query{TaskID: &id}, nil, 1},
		{"por bug", Query{BugID: &id}, nil, 1},
		{"sin filtro", Query{}, domain.ErrListModeInvalid, 0},
		{"dos filtros", Query{StageID: &id, BugID: &id}, domain.ErrListModeInvalid, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{plans: []domain.Plan{plan}}
			items, err := New(reader).Handle(context.Background(), tt.query)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if reader.calls != tt.calls {
				t.Errorf("lecturas = %d", reader.calls)
			}
			if tt.want == nil && (len(items) != 1 || items[0].Version != 2 || items[0].Scope != "S") {
				t.Errorf("items = %+v", items)
			}
		})
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name  string
		query string
		code  int
	}{
		{"por etapa", "?stage_id=" + uuid.NewString(), http.StatusOK},
		{"por tarea", "?task_id=" + uuid.NewString(), http.StatusOK},
		{"por bug", "?bug_id=" + uuid.NewString(), http.StatusOK},
		{"sin filtro", "", http.StatusBadRequest},
		{"id inválido", "?bug_id=abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dev-plans"+tt.query, nil))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
