package list

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
)

type fakeReader struct {
	findings []domain.Finding
	pending  bool
	taskIDs  []uuid.UUID
}

func (r *fakeReader) List(_ context.Context, _ domain.ItemKind, _ uuid.UUID, pending bool) ([]domain.Finding, error) {
	r.pending = pending
	return r.findings, nil
}

// ListByTasks imita a la base: solo los de esas tareas y, con pending, los abiertos
// con los bloqueantes primero.
func (r *fakeReader) ListByTasks(_ context.Context, taskIDs []uuid.UUID, pending bool) ([]domain.Finding, error) {
	r.pending, r.taskIDs = pending, taskIDs
	found := make([]domain.Finding, 0, len(r.findings))
	for _, f := range r.findings {
		if slices.Contains(taskIDs, f.ItemID) && (!pending || f.Status == domain.StatusOpen) {
			found = append(found, f)
		}
	}
	if pending {
		slices.SortStableFunc(found, func(a, b domain.Finding) int {
			switch {
			case a.Blocking == b.Blocking:
				return 0
			case a.Blocking:
				return -1
			}
			return 1
		})
	}
	return found, nil
}

// fakeTasks responde con las tareas de la etapa, retiradas incluidas.
type fakeTasks struct {
	ids []uuid.UUID
	err error
}

func (f fakeTasks) TaskIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return f.ids, f.err
}

func TestHandleListsByItem(t *testing.T) {
	reader := &fakeReader{findings: []domain.Finding{{ID: uuid.New()}}}

	items, err := New(reader, fakeTasks{}).Handle(context.Background(), Query{ItemKind: domain.ItemTask, ItemID: uuid.New(), Pending: true})

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !reader.pending {
		t.Errorf("items = %+v, pending = %v", items, reader.pending)
	}
}

func TestHandleListsByStage(t *testing.T) {
	active, retired, foreign := uuid.New(), uuid.New(), uuid.New()
	reader := &fakeReader{findings: []domain.Finding{
		{ID: uuid.New(), ItemKind: domain.ItemTask, ItemID: active, Status: domain.StatusOpen},
		{ID: uuid.New(), ItemKind: domain.ItemTask, ItemID: retired, Status: domain.StatusOpen, Blocking: true},
		{ID: uuid.New(), ItemKind: domain.ItemTask, ItemID: active, Status: domain.StatusFixed, Blocking: true},
		{ID: uuid.New(), ItemKind: domain.ItemTask, ItemID: foreign, Status: domain.StatusOpen},
	}}
	tasks := fakeTasks{ids: []uuid.UUID{active, retired}}

	all, err := New(reader, tasks).Handle(context.Background(), Query{StageID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ItemID != active || all[1].ItemID != retired {
		t.Errorf("todos = %+v", all)
	}

	pending, err := New(reader, tasks).Handle(context.Background(), Query{StageID: uuid.New(), Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || !pending[0].Blocking || pending[0].ItemID != retired {
		t.Errorf("pendientes = %+v", pending)
	}
}

func TestHandleStageWithoutTasksIsEmpty(t *testing.T) {
	items, err := New(&fakeReader{}, fakeTasks{}).Handle(context.Background(), Query{StageID: uuid.New()})

	if err != nil || items == nil || len(items) != 0 {
		t.Errorf("items = %v, err = %v", items, err)
	}
}

func TestHandlePropagatesTaskError(t *testing.T) {
	want := errors.New("falla")

	_, err := New(&fakeReader{}, fakeTasks{err: want}).Handle(context.Background(), Query{StageID: uuid.New()})

	if !errors.Is(err, want) {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleEmptyIsNotNil(t *testing.T) {
	items, err := New(&fakeReader{}, fakeTasks{}).Handle(context.Background(), Query{ItemKind: domain.ItemBug, ItemID: uuid.New()})

	if err != nil || items == nil || len(items) != 0 {
		t.Errorf("items = %v, err = %v", items, err)
	}
}

func TestHandleRejectsUnknownKind(t *testing.T) {
	_, err := New(&fakeReader{}, fakeTasks{}).Handle(context.Background(), Query{ItemKind: "stage", ItemID: uuid.New()})

	if !errors.Is(err, domain.ErrItemKindInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleRejectsInvalidModes(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name  string
		query Query
	}{
		{"ninguno", Query{}},
		{"solo pending", Query{Pending: true}},
		{"los dos", Query{ItemKind: domain.ItemTask, ItemID: id, StageID: id}},
		{"solo tipo", Query{ItemKind: domain.ItemTask}},
		{"solo elemento", Query{ItemID: id}},
		{"etapa y tipo", Query{ItemKind: domain.ItemTask, StageID: id}},
		{"etapa y elemento", Query{ItemID: id, StageID: id}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&fakeReader{}, fakeTasks{}).Handle(context.Background(), tt.query)

			if !errors.Is(err, domain.ErrListModeInvalid) {
				t.Fatalf("err = %v", err)
			}
		})
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
		{"por etapa", "?stage_id=" + id, http.StatusOK},
		{"pendientes por etapa", "?stage_id=" + id + "&pending=true", http.StatusOK},
		{"sin modo", "", http.StatusBadRequest},
		{"los dos modos", "?item_kind=task&item_id=" + id + "&stage_id=" + id, http.StatusBadRequest},
		{"sin elemento", "?item_kind=task", http.StatusBadRequest},
		{"sin tipo", "?item_id=" + id, http.StatusBadRequest},
		{"elemento inválido", "?item_kind=task&item_id=abc", http.StatusBadRequest},
		{"etapa inválida", "?stage_id=abc", http.StatusBadRequest},
		{"pending inválido", "?item_kind=task&item_id=" + id + "&pending=quizas", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeReader{}, fakeTasks{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/findings"+tt.query, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
