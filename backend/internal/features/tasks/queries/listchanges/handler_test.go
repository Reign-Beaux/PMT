package listchanges

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
	histories []domain.History
	filter    *uuid.UUID
}

func (r *fakeReader) List(_ context.Context, _ uuid.UUID, taskID *uuid.UUID) ([]domain.History, error) {
	r.filter = taskID
	return r.histories, nil
}

func TestHandleReturnsOriginalAndChanges(t *testing.T) {
	id := uuid.New()
	reader := &fakeReader{histories: []domain.History{{
		TaskID: id,
		Title:  "Crear login",
		Changes: []domain.Change{
			{TaskID: id, Version: 1, Kind: domain.ChangeEdit, Title: "Crear login con Google", Reason: "Incongruencia"},
			{TaskID: id, Version: 2, Kind: domain.ChangeRetire, Title: "Crear login con Google", Reason: "No se pensó bien"},
		},
	}}}

	items, err := New(reader).Handle(context.Background(), uuid.New(), &id)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].TaskID != id || items[0].Original.Title != "Crear login" {
		t.Fatalf("items = %+v", items)
	}
	changes := items[0].Changes
	if len(changes) != 2 || changes[1].Kind != domain.ChangeRetire || changes[0].Reason != "Incongruencia" {
		t.Errorf("cambios = %+v", changes)
	}
	if reader.filter == nil || *reader.filter != id {
		t.Errorf("filtro = %v", reader.filter)
	}
}

func TestREST(t *testing.T) {
	path := "/api/stages/" + uuid.NewString() + "/task-changes"
	tests := []struct {
		name   string
		path   string
		status int
		body   string
	}{
		{"sin cambios", path, http.StatusOK, "[]"},
		{"con filtro", path + "?task_id=" + uuid.NewString(), http.StatusOK, "[]"},
		{"filtro inválido", path + "?task_id=abc", http.StatusBadRequest, "inválido"},
		{"padre inválido", "/api/stages/abc/task-changes", http.StatusBadRequest, "inválido"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.body) {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
