package list

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/activity/domain"
)

type fakeReader struct {
	comments []domain.Comment
	kind     domain.ItemKind
	itemID   uuid.UUID
	calls    int
}

func (r *fakeReader) List(_ context.Context, kind domain.ItemKind, itemID uuid.UUID) ([]domain.Comment, error) {
	r.calls++
	r.kind, r.itemID = kind, itemID
	return r.comments, nil
}

func TestHandleLists(t *testing.T) {
	id := uuid.New()
	reader := &fakeReader{comments: []domain.Comment{{Body: "Primero"}, {Body: "Segundo"}}}

	items, err := New(reader).Handle(context.Background(), Query{ItemKind: domain.ItemStage, ItemID: id})

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Body != "Primero" || items[1].Body != "Segundo" {
		t.Errorf("items = %+v", items)
	}
	if reader.kind != domain.ItemStage || reader.itemID != id {
		t.Errorf("consulta = %s %s", reader.kind, reader.itemID)
	}
}

func TestHandleEmpty(t *testing.T) {
	items, err := New(&fakeReader{}).Handle(context.Background(), Query{ItemKind: domain.ItemBug, ItemID: uuid.New()})

	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Errorf("items = %#v", items)
	}
}

func TestHandleRejectsUnknownKind(t *testing.T) {
	reader := &fakeReader{}

	_, err := New(reader).Handle(context.Background(), Query{ItemKind: "finding", ItemID: uuid.New()})

	if !errors.Is(err, domain.ErrItemKindInvalid) {
		t.Errorf("error = %v", err)
	}
	if reader.calls != 0 {
		t.Errorf("llamadas = %d", reader.calls)
	}
}

func TestREST(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name   string
		target string
		status int
	}{
		{"válido", "/api/comments?item_kind=task&item_id=" + id, http.StatusOK},
		{"tipo desconocido", "/api/comments?item_kind=finding&item_id=" + id, http.StatusBadRequest},
		{"sin tipo", "/api/comments?item_id=" + id, http.StatusBadRequest},
		{"sin id", "/api/comments?item_kind=task", http.StatusBadRequest},
		{"id inválido", "/api/comments?item_kind=task&item_id=abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTEmptyIsArray(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeReader{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/comments?item_kind=bug&item_id="+uuid.NewString(), nil))

	var items []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil || items == nil {
		t.Errorf("cuerpo = %s, error = %v", rec.Body, err)
	}
}
