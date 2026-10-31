package revise

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
	current domain.Task
	changes []domain.Change
}

func newRepo() *fakeRepo {
	return &fakeRepo{current: domain.Task{ID: uuid.New(), Title: "Crear login"}}
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	if id != r.current.ID {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return r.current, nil
}

func (r *fakeRepo) Revise(_ context.Context, change domain.Change) (domain.Task, domain.Change, error) {
	change.Version = 2
	r.current.Title = change.Title
	r.current.Description = change.Description
	r.changes = append(r.changes, change)
	return r.current, change, nil
}

func TestHandleSavesNewVersion(t *testing.T) {
	repo := newRepo()

	result, err := New(repo, fakeStages{}).Handle(context.Background(), Command{
		ID: repo.current.ID, Title: "Crear login con Google", Reason: "Incongruencia", RequestedBy: "PM",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Crear login con Google" || result.Version != 2 {
		t.Errorf("result = %+v", result)
	}
	if len(repo.changes) != 1 || repo.changes[0].Kind != domain.ChangeEdit || repo.changes[0].Title != "Crear login con Google" {
		t.Errorf("cambios = %+v", repo.changes)
	}
}

func TestHandleKeepsCurrentWhenInvalid(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want error
	}{
		{"sin title", Command{Reason: "M", RequestedBy: "PM"}, domain.ErrTaskTitleRequired},
		{"sin motivo", Command{Title: "A", RequestedBy: "PM"}, domain.ErrChangeReasonRequired},
		{"sin solicitante", Command{Title: "A", Reason: "M"}, domain.ErrChangeRequesterRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()
			tt.cmd.ID = repo.current.ID

			_, err := New(repo, fakeStages{}).Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.changes) != 0 || repo.current.Title != "Crear login" {
				t.Errorf("repo = %+v", repo)
			}
		})
	}
}

func TestHandleMissingOrRetired(t *testing.T) {
	repo := newRepo()

	_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: uuid.New(), Title: "A", Reason: "M", RequestedBy: "PM"})

	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestREST(t *testing.T) {
	valid := `{"title":"A","reason":"M","requested_by":"PM"}`
	tests := []struct {
		name   string
		id     func(uuid.UUID) string
		body   string
		status int
	}{
		{"válida", uuid.UUID.String, valid, http.StatusOK},
		{"sin motivo", uuid.UUID.String, `{"title":"A","requested_by":"PM"}`, http.StatusBadRequest},
		{"id inválido", func(uuid.UUID) string { return "abc" }, valid, http.StatusBadRequest},
		{"no existe", func(uuid.UUID) string { return uuid.NewString() }, valid, http.StatusNotFound},
		{"status no editable", uuid.UUID.String, `{"title":"A","status":"done"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()
			mux := http.NewServeMux()
			New(repo, fakeStages{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/tasks/"+tt.id(repo.current.ID)+"/revise", strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandleRejectsStageInOwnPlanning(t *testing.T) {
	errOwnPlanning := kernel.Conflict("la etapa está en su propio planning")
	repo := newRepo()

	_, err := New(repo, fakeStages{err: errOwnPlanning}).Handle(context.Background(), Command{
		ID: repo.current.ID, Title: "A", Reason: "M", RequestedBy: "PM",
	})

	if !errors.Is(err, errOwnPlanning) || len(repo.changes) != 0 {
		t.Errorf("err = %v, cambios = %+v", err, repo.changes)
	}
}
