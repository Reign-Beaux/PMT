package listchanges

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
	histories []domain.History
	filter    *uuid.UUID
}

func (r *fakeReader) List(_ context.Context, _ uuid.UUID, stageID *uuid.UUID) ([]domain.History, error) {
	r.filter = stageID
	return r.histories, nil
}

func TestHandleReturnsOriginalAndChanges(t *testing.T) {
	id := uuid.New()
	reader := &fakeReader{histories: []domain.History{{
		StageID: id,
		Name:    "Crear login",
		Changes: []domain.Change{
			{StageID: id, Version: 1, Kind: domain.ChangeEdit, Name: "Crear login con Google", Reason: "Incongruencia"},
			{StageID: id, Version: 2, Kind: domain.ChangeRetire, Name: "Crear login con Google", Reason: "No se pensó bien"},
		},
	}}}

	items, err := New(reader).Handle(context.Background(), uuid.New(), &id)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].StageID != id || items[0].Original.Name != "Crear login" {
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
	path := "/api/projects/" + uuid.NewString() + "/stage-changes"
	tests := []struct {
		name   string
		path   string
		status int
		body   string
	}{
		{"sin cambios", path, http.StatusOK, "[]"},
		{"con filtro", path + "?stage_id=" + uuid.NewString(), http.StatusOK, "[]"},
		{"filtro inválido", path + "?stage_id=abc", http.StatusBadRequest, "inválido"},
		{"padre inválido", "/api/projects/abc/stage-changes", http.StatusBadRequest, "inválido"},
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
