package delete

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
)

type fakeRepo struct {
	existing uuid.UUID
}

func (r *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if id != r.existing {
		return domain.ErrStageNotFound
	}
	return nil
}

func TestREST(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name   string
		id     string
		status int
	}{
		{"existe", id.String(), http.StatusNoContent},
		{"id inválido", "abc", http.StatusBadRequest},
		{"no existe", uuid.NewString(), http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{existing: id}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/stages/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
