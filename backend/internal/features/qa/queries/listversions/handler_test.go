package listversions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
)

type fakeReader struct{ versions []domain.Version }

func (f *fakeReader) ListVersions(context.Context, uuid.UUID) ([]domain.Version, error) {
	return f.versions, nil
}

func TestHandle(t *testing.T) {
	reader := &fakeReader{versions: []domain.Version{{Version: 1}, {Version: 2, Reason: "M"}}}
	items, err := New(reader).Handle(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Reason != "M" {
		t.Errorf("items = %+v", items)
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name string
		id   string
		code int
	}{
		{"sin versiones", uuid.NewString(), http.StatusOK},
		{"id inválido", "abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/test-plans/"+tt.id+"/versions", nil))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
