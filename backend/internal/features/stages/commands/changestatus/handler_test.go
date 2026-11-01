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

	"pmt/internal/features/stages/domain"
	"pmt/internal/platform/rest"
)

type fakeRepo struct {
	stages   map[uuid.UUID]domain.Stage
	updates  int
	recorded []domain.StatusChange
	changes  []domain.Change
}

func newRepoWith(t *testing.T, status domain.Status) (*fakeRepo, uuid.UUID) {
	t.Helper()
	s, err := domain.NewStage(uuid.New(), "Diseño", "Inicial")
	s.Lead = "tl"
	if err != nil {
		t.Fatal(err)
	}
	s.Status = status
	return &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}, s.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	s, ok := r.stages[id]
	if !ok {
		return domain.Stage{}, domain.ErrStageNotFound
	}
	return s, nil
}

func (r *fakeRepo) ChangeStatus(_ context.Context, s domain.Stage, change domain.StatusChange) (domain.Stage, error) {
	r.updates++
	r.recorded = append(r.recorded, change)
	r.stages[s.ID] = s
	return s, nil
}

func (r *fakeRepo) LeavePlanning(_ context.Context, s domain.Stage, change domain.Change, statusChange domain.StatusChange) (domain.Stage, error) {
	r.recorded = append(r.recorded, statusChange)
	r.changes = append(r.changes, change)
	s.AddReason, s.AddRequestedBy = "", ""
	r.stages[s.ID] = s
	return s, nil
}

// fakeTasks responde como si la etapa tuviera tareas en ese estado. Una etapa
// sin tareas responde true, igual que una con todas terminadas.
type fakeTasks struct {
	allDone    bool
	err        error
	calls      int
	releasable error
	checks     int
}

func (f *fakeTasks) AllDone(context.Context, uuid.UUID) (bool, error) {
	f.calls++
	return f.allDone, f.err
}

func (f *fakeTasks) EnsureReleasable(context.Context, uuid.UUID) error {
	f.checks++
	return f.releasable
}

func TestHandleChangesStatus(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusPlanning)
	tasks := &fakeTasks{}

	result, err := New(repo, tasks, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: actorFor(domain.StatusStandBy), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusStandBy || repo.stages[id].Status != domain.StatusStandBy {
		t.Errorf("result = %+v", result)
	}
	if tasks.calls != 0 {
		t.Errorf("consultó las tareas %d veces", tasks.calls)
	}
}

func TestHandleRejectedStatusIsNotSaved(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		want   error
	}{
		{"transición no permitida", domain.StatusDone, domain.ErrStageStatusTransition},
		{"mismo status", domain.StatusPlanning, domain.ErrStageStatusTransition},
		{"desconocido", "archived", domain.ErrStageStatusInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, domain.StatusPlanning)
			tasks := &fakeTasks{allDone: true}

			_, err := New(repo, tasks, workingProject).Handle(context.Background(), Command{ID: id, Status: tt.status, Actor: actorFor(tt.status), Reason: "motivo"})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if repo.updates != 0 || repo.stages[id].Status != domain.StatusPlanning {
				t.Errorf("updates = %d, etapa = %+v", repo.updates, repo.stages[id])
			}
			if tasks.calls != 0 {
				t.Errorf("consultó las tareas %d veces", tasks.calls)
			}
		})
	}
}

func TestHandleMissingStage(t *testing.T) {
	repo, _ := newRepoWith(t, domain.StatusPlanning)

	_, err := New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: uuid.New(), Status: domain.StatusStandBy, Actor: actorFor(domain.StatusStandBy), Reason: "motivo"})

	if !errors.Is(err, domain.ErrStageNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleClosesStageWhenAllTasksDone(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusInDev)
	tasks := &fakeTasks{allDone: true}

	result, err := New(repo, tasks, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusDone, Actor: actorFor(domain.StatusDone), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusDone || repo.stages[id].Status != domain.StatusDone || tasks.calls != 1 {
		t.Errorf("result = %+v, consultas = %d", result, tasks.calls)
	}
}

func TestHandleKeepsStageOpenWithPendingTasks(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusInDev)

	_, err := New(repo, &fakeTasks{allDone: false}, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusDone, Actor: actorFor(domain.StatusDone), Reason: "motivo"})

	if !errors.Is(err, domain.ErrStageTasksPending) {
		t.Fatalf("err = %v", err)
	}
	if repo.updates != 0 || repo.stages[id].Status != domain.StatusInDev {
		t.Errorf("updates = %d, etapa = %+v", repo.updates, repo.stages[id])
	}
}

func TestHandleKeepsStageOpenWhenCheckFails(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusInDev)
	failure := errors.New("fallo de lectura")

	_, err := New(repo, &fakeTasks{allDone: true, err: failure}, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusDone, Actor: actorFor(domain.StatusDone), Reason: "motivo"})

	if !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
	if repo.updates != 0 {
		t.Error("no debió guardar el status")
	}
}

func TestHandleReopenDoesNotCheckTasks(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusDone)
	tasks := &fakeTasks{allDone: false}

	result, err := New(repo, tasks, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusInDev, Actor: actorFor(domain.StatusInDev), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusInDev || tasks.calls != 0 {
		t.Errorf("result = %+v, consultas = %d", result, tasks.calls)
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.Status
		allDone bool
		id      func(uuid.UUID) string
		body    string
		status  int
	}{
		{"válido", domain.StatusPlanning, true, uuid.UUID.String, `{"status":"stand_by"}`, http.StatusOK},
		{"id inválido", domain.StatusPlanning, true, func(uuid.UUID) string { return "abc" }, `{"status":"stand_by"}`, http.StatusBadRequest},
		{"no existe", domain.StatusPlanning, true, func(uuid.UUID) string { return uuid.NewString() }, `{"status":"stand_by"}`, http.StatusNotFound},
		{"status desconocido", domain.StatusPlanning, true, uuid.UUID.String, `{"status":"archived"}`, http.StatusBadRequest},
		{"sin status", domain.StatusPlanning, true, uuid.UUID.String, `{}`, http.StatusBadRequest},
		{"campo desconocido", domain.StatusPlanning, true, uuid.UUID.String, `{"status":"stand_by","name":"Otro"}`, http.StatusBadRequest},
		{"transición no permitida", domain.StatusPlanning, true, uuid.UUID.String, `{"status":"done"}`, http.StatusConflict},
		{"mismo status", domain.StatusPlanning, true, uuid.UUID.String, `{"status":"planning"}`, http.StatusConflict},
		{"cierre con tareas pendientes", domain.StatusInDev, false, uuid.UUID.String, `{"status":"done"}`, http.StatusConflict},
		{"cierre", domain.StatusInDev, true, uuid.UUID.String, `{"status":"done"}`, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)
			mux := http.NewServeMux()
			New(repo, &fakeTasks{allDone: tt.allDone}, workingProject).Register(mux)
			rec := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodPut, "/api/stages/"+tt.id(id)+"/status", strings.NewReader(tt.body))
			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), actorFor(tt.from))))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTReturnsWholeStage(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusPlanning)
	mux := http.NewServeMux()
	New(repo, &fakeTasks{}, workingProject).Register(mux)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/stages/"+id.String()+"/status", strings.NewReader(`{"status":"stand_by"}`)))

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"id", "project_id", "name", "description", "position", "created_at", "updated_at"} {
		if _, ok := body[field]; !ok {
			t.Errorf("falta %s en %s", field, rec.Body)
		}
	}
	if body["status"] != "stand_by" {
		t.Errorf("cuerpo = %s", rec.Body)
	}
}

func TestHandleWritesAddWhenLeavingOwnPlanning(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusStandBy, domain.StatusStopped, domain.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			s, err := domain.NewOwnPlanningStage(uuid.New(), "Pagos", "Inicial", "Petición", "PM")
			if err != nil {
				t.Fatal(err)
			}
			s.Name = "Cobros"
			repo := &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}

			result, err := New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: s.ID, Status: status, Actor: actorFor(status), Reason: "motivo"})

			if err != nil {
				t.Fatal(err)
			}
			if len(repo.changes) != 1 {
				t.Fatalf("cambios = %+v", repo.changes)
			}
			change := repo.changes[0]
			if change.Kind != domain.ChangeAdd || change.Name != "Cobros" || change.Reason != "Petición" || change.RequestedBy != "PM" {
				t.Errorf("cambio = %+v", change)
			}
			if result.Status != status || result.AddReason != "" {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestHandleDoesNotReopenOwnPlanning(t *testing.T) {
	s, err := domain.NewOwnPlanningStage(uuid.New(), "Pagos", "Inicial", "Petición", "PM")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}
	handler := New(repo, &fakeTasks{}, workingProject)

	if _, err := handler.Handle(context.Background(), Command{ID: s.ID, Status: domain.StatusStandBy, Actor: actorFor(domain.StatusStandBy), Reason: "motivo"}); err != nil {
		t.Fatal(err)
	}
	result, err := handler.Handle(context.Background(), Command{ID: s.ID, Status: domain.StatusPlanning, Actor: actorFor(domain.StatusPlanning), Reason: "motivo"})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusPlanning || repo.stages[s.ID].InOwnPlanning() || len(repo.changes) != 1 {
		t.Errorf("result = %+v, cambios = %+v", result, repo.changes)
	}
}

func TestHandleDoesNotWriteAddForOtherStages(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusPlanning)

	if _, err := New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Actor: actorFor(domain.StatusStandBy), Reason: "motivo"}); err != nil {
		t.Fatal(err)
	}
	if len(repo.changes) != 0 {
		t.Errorf("cambios = %+v", repo.changes)
	}
}
