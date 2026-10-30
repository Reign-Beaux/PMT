package get

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

type fakeReader struct {
	stage domain.Stage
}

func (r *fakeReader) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	if id != r.stage.ID {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	return r.stage, nil
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
			New(&fakeReader{stage: domain.Stage{ID: id}}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stages/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
