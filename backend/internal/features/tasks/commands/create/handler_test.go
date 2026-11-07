package create

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

var errNotPlanning = kernel.Conflict("el proyecto no está en planning")

type fakeStages struct {
	err error
}

func (c fakeStages) EnsureTasksEditable(context.Context, uuid.UUID) error {
	return c.err
}

type fakeRepo struct {
	saved []domain.Task
	err   error
}

func (r *fakeRepo) Create(_ context.Context, t domain.Task) (domain.Task, error) {
	if r.err != nil {
		return domain.Task{}, r.err
	}
	r.saved = append(r.saved, t)
	return t, nil
}

func TestHandleCreatesTask(t *testing.T) {
	repo := &fakeRepo{}
	stageID := uuid.New()

	result, err := New(repo, fakeStages{}).Handle(context.Background(), Command{Actor: kernel.Owner, StageID: stageID, Title: " Revisar "})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Revisar" || result.StageID != stageID || len(repo.saved) != 1 {
		t.Errorf("result = %+v, guardadas = %d", result, len(repo.saved))
	}
	if result.Status != domain.StatusBacklog {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleRejectsInvalidTask(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{Actor: kernel.Owner, StageID: uuid.New(), Title: " "})

	if !errors.Is(err, domain.ErrTaskTitleRequired) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar la tarea")
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeStages{err: errNotPlanning}).Handle(context.Background(), Command{Actor: kernel.Owner, StageID: uuid.New(), Title: "Revisar"})

	if !errors.Is(err, errNotPlanning) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar")
	}
}

func TestRESTRejectsOutsidePlanning(t *testing.T) {
	mux := http.NewServeMux()
	New(&fakeRepo{}, fakeStages{err: errNotPlanning}).Register(mux)
	rec := httptest.NewRecorder()
	path := "/api/stages/" + uuid.NewString() + "/tasks"

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"title":"A"}`)))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}

func TestREST(t *testing.T) {
	path := "/api/stages/" + uuid.NewString() + "/tasks"
	tests := []struct {
		name   string
		path   string
		body   string
		err    error
		status int
	}{
		{"válida", path, `{"title":"Revisar"}`, nil, http.StatusCreated},
		{"título vacío", path, `{"title":" "}`, nil, http.StatusBadRequest},
		{"json inválido", path, `{`, nil, http.StatusBadRequest},
		{"status no editable", path, `{"title":"A","status":"ready"}`, nil, http.StatusBadRequest},
		{"blocked desconocido", path, `{"title":"A","blocked":true}`, nil, http.StatusBadRequest},
		{"etapa inválida", "/api/stages/abc/tasks", `{"title":"A"}`, nil, http.StatusBadRequest},
		{"etapa inexistente", path, `{"title":"A"}`, domain.ErrStageNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{err: tt.err}, fakeStages{}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
