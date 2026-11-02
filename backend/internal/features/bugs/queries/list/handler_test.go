package list

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

type fakeReader struct {
	bugs []domain.Bug
}

func (r fakeReader) List(context.Context, uuid.UUID) ([]domain.Bug, error) {
	return r.bugs, nil
}

func TestHandleListsBugs(t *testing.T) {
	items, err := New(fakeReader{bugs: []domain.Bug{{Title: "A"}, {Title: "B"}}}).Handle(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Title != "B" {
		t.Errorf("items = %+v", items)
	}
}

func TestRESTReturnsEmptyList(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeReader{}).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects/"+uuid.NewString()+"/bugs", nil))

	var items []Item
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || items == nil || len(items) != 0 {
		t.Errorf("código = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
