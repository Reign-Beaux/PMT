package list

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
)

type fakeReader struct {
	plans []domain.Plan
	task  *uuid.UUID
}

func (f *fakeReader) List(_ context.Context, _ uuid.UUID, taskID *uuid.UUID) ([]domain.Plan, error) {
	f.task = taskID
	return f.plans, nil
}

func TestHandle(t *testing.T) {
	plan := domain.Plan{ID: uuid.New(), Status: domain.StatusApproved, Latest: domain.Version{Version: 2, HappyPath: "H"}}
	items, err := New(&fakeReader{plans: []domain.Plan{plan}}).Handle(context.Background(), uuid.New(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Version != 2 || items[0].HappyPath != "H" {
		t.Errorf("items = %+v", items)
	}
}

func TestREST(t *testing.T) {
	stage := uuid.NewString()
	tests := []struct {
		name  string
		query string
		code  int
	}{
		{"sin filtro", "", http.StatusOK},
		{"con tarea", "?task_id=" + uuid.NewString(), http.StatusOK},
		{"tarea inválida", "?task_id=abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stages/"+stage+"/test-plans"+tt.query, nil))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
