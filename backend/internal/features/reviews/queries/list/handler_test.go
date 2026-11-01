package list

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
)

type fakeReader struct {
	findings []domain.Finding
	pending  bool
}

func (r *fakeReader) List(_ context.Context, _ domain.ItemKind, _ uuid.UUID, pending bool) ([]domain.Finding, error) {
	r.pending = pending
	return r.findings, nil
}

func TestHandleLists(t *testing.T) {
	reader := &fakeReader{findings: []domain.Finding{{ID: uuid.New()}}}

	items, err := New(reader).Handle(context.Background(), domain.ItemTask, uuid.New(), true)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !reader.pending {
		t.Errorf("items = %+v, pending = %v", items, reader.pending)
	}
}

func TestHandleEmptyIsNotNil(t *testing.T) {
	items, err := New(&fakeReader{}).Handle(context.Background(), domain.ItemBug, uuid.New(), false)

	if err != nil || items == nil || len(items) != 0 {
		t.Errorf("items = %v, err = %v", items, err)
	}
}

func TestHandleRejectsUnknownKind(t *testing.T) {
	_, err := New(&fakeReader{}).Handle(context.Background(), "stage", uuid.New(), false)

	if !errors.Is(err, domain.ErrItemKindInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestREST(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name   string
		query  string
		status int
	}{
		{"válido", "?item_kind=task&item_id=" + id, http.StatusOK},
		{"pendientes", "?item_kind=bug&item_id=" + id + "&pending=true", http.StatusOK},
		{"sin elemento", "?item_kind=task", http.StatusBadRequest},
		{"sin tipo", "?item_id=" + id, http.StatusBadRequest},
		{"pending inválido", "?item_kind=task&item_id=" + id + "&pending=quizas", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/findings"+tt.query, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
