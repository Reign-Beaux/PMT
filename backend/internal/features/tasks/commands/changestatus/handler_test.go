package changestatus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/platform/rest"
)

// fakeRepo imita al store: Get no encuentra lo retirado y ambas lecturas
// devuelven la última versión, no el contenido original de la fila.
type fakeRepo struct {
	tasks    map[uuid.UUID]domain.Task
	retired  map[uuid.UUID]bool
	latest   map[uuid.UUID]domain.Task
	updates  int
	recorded []domain.StatusChange
	onChange func(domain.Task)
}

func newRepoWith(t *testing.T, status domain.Status) (*fakeRepo, uuid.UUID) {
	t.Helper()
	task, err := domain.NewTask(uuid.New(), "Login", "Inicial")
	if err != nil {
		t.Fatal(err)
	}
	task.Status = status
	return &fakeRepo{
		tasks:   map[uuid.UUID]domain.Task{task.ID: task},
		retired: map[uuid.UUID]bool{},
		latest:  map[uuid.UUID]domain.Task{},
	}, task.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	task, ok := r.tasks[id]
	if !ok || r.retired[id] {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return r.withLatest(task), nil
}

func (r *fakeRepo) ChangeStatus(_ context.Context, task domain.Task, change domain.StatusChange) (domain.Task, error) {
	r.updates++
	r.recorded = append(r.recorded, change)
	if r.onChange != nil {
		r.onChange(task)
	}
	row := r.tasks[task.ID]
	row.Status = task.Status
	r.tasks[task.ID] = row
	return r.withLatest(row), nil
}

func (r *fakeRepo) withLatest(task domain.Task) domain.Task {
	if latest, ok := r.latest[task.ID]; ok {
		task.Title, task.Description = latest.Title, latest.Description
	}
	return task
}

func TestHandleChangesStatus(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusBacklog)

	result, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Status: domain.StatusReady, Actor: actorFor(domain.StatusReady), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusReady || repo.tasks[id].Status != domain.StatusReady {
		t.Errorf("result = %+v", result)
	}
}

func TestHandleReturnsLatestVersion(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusBacklog)
	repo.latest[id] = domain.Task{Title: "Login v2", Description: "Revisada"}

	result, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Status: domain.StatusReady, Actor: actorFor(domain.StatusReady), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Login v2" || result.Description != "Revisada" || result.Status != domain.StatusReady {
		t.Errorf("result = %+v", result)
	}
	if row := repo.tasks[id]; row.Title != "Login" || row.Description != "Inicial" {
		t.Errorf("la fila conserva el original: %+v", row)
	}
}

func TestHandleRejectedStatusIsNotSaved(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		want   error
	}{
		{"transición no permitida", domain.StatusDone, domain.ErrTaskStatusTransition},
		{"mismo status", domain.StatusBacklog, domain.ErrTaskStatusTransition},
		{"status desconocido", "archived", domain.ErrTaskStatusInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, domain.StatusBacklog)

			_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Status: tt.status, Actor: actorFor(tt.status), Reason: "motivo"})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if repo.updates != 0 || repo.tasks[id].Status != domain.StatusBacklog {
				t.Errorf("no debió guardar: %+v", repo.tasks[id])
			}
		})
	}
}

func TestHandleNotFound(t *testing.T) {
	t.Run("inexistente", func(t *testing.T) {
		repo, _ := newRepoWith(t, domain.StatusBacklog)

		_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: uuid.New(), Status: domain.StatusReady, Actor: actorFor(domain.StatusReady), Reason: "motivo"})

		if !errors.Is(err, domain.ErrTaskNotFound) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("retirada", func(t *testing.T) {
		repo, id := newRepoWith(t, domain.StatusBacklog)
		repo.retired[id] = true

		_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Status: domain.StatusReady, Actor: actorFor(domain.StatusReady), Reason: "motivo"})

		if !errors.Is(err, domain.ErrTaskNotFound) {
			t.Fatalf("err = %v", err)
		}
		if repo.updates != 0 {
			t.Error("no debió guardar el status")
		}
	})
}

func TestRest(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		path func(uuid.UUID) string
		body string
		want int
	}{
		{"cambia", domain.StatusBacklog, taskPath, `{"status":"ready"}`, http.StatusOK},
		{"id inválido", domain.StatusBacklog, func(uuid.UUID) string { return "/api/tasks/abc/status" }, `{"status":"ready"}`, http.StatusBadRequest},
		{"campo desconocido", domain.StatusBacklog, taskPath, `{"status":"ready","x":1}`, http.StatusBadRequest},
		{"status desconocido", domain.StatusBacklog, taskPath, `{"status":"archived"}`, http.StatusBadRequest},
		{"transición no permitida", domain.StatusDone, taskPath, `{"status":"backlog"}`, http.StatusConflict},
		{"inexistente", domain.StatusBacklog, func(uuid.UUID) string { return "/api/tasks/" + uuid.NewString() + "/status" }, `{"status":"ready"}`, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)
			mux := http.NewServeMux()
			New(repo, fakeStages{}).Register(mux)

			req := httptest.NewRequest(http.MethodPut, tt.path(id), strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), actorFor(""))))

			if rec.Code != tt.want {
				t.Fatalf("código = %d, cuerpo = %s", rec.Code, rec.Body)
			}
			if tt.want != http.StatusOK {
				return
			}
			var result Result
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.ID != id || result.Status != domain.StatusReady || result.Title != "Login" {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func taskPath(id uuid.UUID) string {
	return "/api/tasks/" + id.String() + "/status"
}
