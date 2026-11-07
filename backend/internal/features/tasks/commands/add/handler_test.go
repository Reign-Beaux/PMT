package add

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

type fakeStages struct {
	err error
}

func (s fakeStages) EnsureHistoryRule(context.Context, uuid.UUID) error {
	return s.err
}

type fakeRepo struct {
	saved   []domain.Task
	changes []domain.Change
	err     error
}

func (r *fakeRepo) Add(_ context.Context, task domain.Task, change domain.Change) (domain.Task, domain.Change, error) {
	if r.err != nil {
		return domain.Task{}, domain.Change{}, r.err
	}
	change.Version = 1
	r.saved = append(r.saved, task)
	r.changes = append(r.changes, change)
	return task, change, nil
}

func newHandler(repo Repository) *Handler {
	return New(repo, fakeStages{})
}

func TestHandleAdds(t *testing.T) {
	repo := &fakeRepo{}
	parent := uuid.New()

	result, err := newHandler(repo).Handle(context.Background(), Command{
		Actor:   kernel.Owner,
		StageID: parent, Title: "Nueva", Reason: "Faltaba", RequestedBy: "PM",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Nueva" || result.StageID != parent || result.Version != 1 {
		t.Errorf("result = %+v", result)
	}
	if len(repo.changes) != 1 || repo.changes[0].Kind != domain.ChangeAdd || repo.changes[0].Reason != "Faltaba" {
		t.Errorf("cambios = %+v", repo.changes)
	}
}

func TestHandleRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want error
	}{
		{"sin title", Command{Actor: kernel.Owner, Reason: "M", RequestedBy: "PM"}, domain.ErrTaskTitleRequired},
		{"sin motivo", Command{Actor: kernel.Owner, Title: "A", RequestedBy: "PM"}, domain.ErrChangeReasonRequired},
		{"sin solicitante", Command{Actor: kernel.Owner, Title: "A", Reason: "M"}, domain.ErrChangeRequesterRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			_, err := newHandler(repo).Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar")
			}
		})
	}
}

func TestHandleRejectsRetiredStage(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeStages{err: domain.ErrStageNotFound}).Handle(context.Background(), Command{
		Actor: kernel.Owner,
		Title: "Nueva", Reason: "Faltaba", RequestedBy: "PM",
	})

	if !errors.Is(err, domain.ErrStageNotFound) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar")
	}
}

func TestREST(t *testing.T) {
	path := "/api/stages/" + uuid.NewString() + "/tasks/add"
	valid := `{"title":"A","reason":"M","requested_by":"PM"}`
	tests := []struct {
		name   string
		path   string
		body   string
		err    error
		status int
	}{
		{"válida", path, valid, nil, http.StatusCreated},
		{"sin motivo", path, `{"title":"A","requested_by":"PM"}`, nil, http.StatusBadRequest},
		{"id inválido", "/api/stages/abc/tasks/add", valid, nil, http.StatusBadRequest},
		{"campo desconocido", path, `{"title":"A","version":2}`, nil, http.StatusBadRequest},
		{"padre inexistente", path, valid, domain.ErrStageNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			newHandler(&fakeRepo{err: tt.err}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
