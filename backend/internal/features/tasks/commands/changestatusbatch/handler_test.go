package changestatusbatch

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
	"pmt/internal/platform/rest"
)

var (
	dev = kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}
	tl  = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	cr  = kernel.Actor{Name: "cr", Profile: kernel.ProfileCR}
)

type fakeRepo struct {
	tasks   []domain.Task
	saved   []domain.Task
	changes []domain.StatusChange
}

// ChangeBatch reproduce al store: entrega lo que existe y guarda solo si decide acepta.
func (r *fakeRepo) ChangeBatch(_ context.Context, ids []uuid.UUID, decide func([]domain.Task) ([]domain.Task, []domain.StatusChange, error)) ([]domain.Task, error) {
	var current []domain.Task
	for _, task := range r.tasks {
		for _, id := range ids {
			if task.ID == id {
				current = append(current, task)
			}
		}
	}
	tasks, changes, err := decide(current)
	if err != nil {
		return nil, err
	}
	r.saved, r.changes = tasks, changes
	return tasks, nil
}

type fakeStages struct {
	team  kernel.Team
	err   error
	calls *int
}

func (f fakeStages) count() {
	if f.calls != nil {
		*f.calls++
	}
}

func (f fakeStages) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	f.count()
	return f.team, nil
}

func (f fakeStages) EnsureWorkOpen(context.Context, uuid.UUID) error {
	f.count()
	return f.err
}

func (f fakeStages) EnsureTaskCancellable(context.Context, uuid.UUID) error {
	f.count()
	return nil
}

var team = kernel.Team{Lead: "tl", Members: []string{"dev"}}

func newTasks(stageID uuid.UUID, statuses ...domain.Status) ([]domain.Task, []uuid.UUID) {
	tasks := make([]domain.Task, 0, len(statuses))
	ids := make([]uuid.UUID, 0, len(statuses))
	for _, status := range statuses {
		task := domain.Task{ID: uuid.New(), StageID: stageID, Status: status}
		tasks = append(tasks, task)
		ids = append(ids, task.ID)
	}
	return tasks, ids
}

func TestHandleTakesAllAndRecordsEach(t *testing.T) {
	stageID := uuid.New()
	tasks, ids := newTasks(stageID, domain.StatusReady, domain.StatusReady)
	repo := &fakeRepo{tasks: tasks}

	items, err := New(repo, fakeStages{team: team}).Handle(context.Background(), Command{StageID: stageID, TaskIDs: ids, Status: domain.StatusInDev, Reason: " toma ", Actor: dev})

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || len(repo.changes) != 2 {
		t.Fatalf("items = %d, registros = %d", len(items), len(repo.changes))
	}
	for i, item := range items {
		if item.Status != domain.StatusInDev || item.Assignee != "dev" {
			t.Errorf("item = %+v", item)
		}
		change := repo.changes[i]
		if change.TaskID != item.ID || change.From != domain.StatusReady || change.To != domain.StatusInDev || change.ActorName != "dev" || change.Reason != "toma" {
			t.Errorf("registro = %+v", change)
		}
	}
}

func TestHandleAcceptsDifferentOrigins(t *testing.T) {
	stageID := uuid.New()
	tasks, ids := newTasks(stageID, domain.StatusBacklog, domain.StatusInDev)
	repo := &fakeRepo{tasks: tasks}

	items, err := New(repo, fakeStages{team: team}).Handle(context.Background(), Command{StageID: stageID, TaskIDs: ids, Status: domain.StatusCancelled, Reason: "fuera", Actor: tl})

	if err != nil || len(items) != 2 || repo.changes[0].From != domain.StatusBacklog || repo.changes[1].From != domain.StatusInDev {
		t.Fatalf("items = %+v, registros = %+v, err = %v", items, repo.changes, err)
	}
}

func TestHandleRejectsWholeBatch(t *testing.T) {
	stageID := uuid.New()
	closed := kernel.Conflict("etapa cerrada")
	other, otherIDs := newTasks(uuid.New(), domain.StatusInReview)
	retired := uuid.New()

	tests := []struct {
		name     string
		statuses []domain.Status
		extra    []uuid.UUID
		extras   []domain.Task
		stages   fakeStages
		status   domain.Status
		reason   string
		actor    kernel.Actor
		want     error
	}{
		{"una sin poder avanzar", []domain.Status{domain.StatusInReview, domain.StatusBacklog}, nil, nil, fakeStages{team: team}, domain.StatusInQA, "", cr, domain.ErrTaskStatusTransition},
		{"perfil incorrecto", []domain.Status{domain.StatusInReview}, nil, nil, fakeStages{team: team}, domain.StatusInQA, "", dev, kernel.ErrTransitionForbidden},
		{"sin motivo", []domain.Status{domain.StatusInReview, domain.StatusInReview}, nil, nil, fakeStages{team: team}, domain.StatusPendingClose, "", cr, kernel.ErrReasonRequired},
		{"regla entre status", []domain.Status{domain.StatusInReview}, nil, nil, fakeStages{team: team, err: closed}, domain.StatusInQA, "", cr, closed},
		{"de otra etapa", []domain.Status{domain.StatusInReview}, otherIDs, other, fakeStages{team: team}, domain.StatusInQA, "", cr, domain.ErrTaskNotFound},
		{"retirada", []domain.Status{domain.StatusInReview}, []uuid.UUID{retired}, nil, fakeStages{team: team}, domain.StatusInQA, "", cr, domain.ErrTaskNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, ids := newTasks(stageID, tt.statuses...)
			repo := &fakeRepo{tasks: append(tasks, tt.extras...)}

			_, err := New(repo, tt.stages).Handle(context.Background(), Command{StageID: stageID, TaskIDs: append(ids, tt.extra...), Status: tt.status, Reason: tt.reason, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if !strings.Contains(err.Error(), "tarea ") {
				t.Errorf("el error debe nombrar la tarea: %v", err)
			}
			if len(repo.saved) != 0 {
				t.Errorf("no debió guardar: %+v", repo.saved)
			}
		})
	}
}

func TestHandleValidatesIDs(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name string
		ids  []uuid.UUID
		want error
	}{
		{"vacía", nil, ErrNoTasks},
		{"repetida", []uuid.UUID{id, id}, ErrRepeatedTask},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&fakeRepo{}, fakeStages{}).Handle(context.Background(), Command{StageID: uuid.New(), TaskIDs: tt.ids, Status: domain.StatusReady, Actor: tl})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestHandleForbiddenDoesNotQueryStages(t *testing.T) {
	stageID := uuid.New()
	tasks, ids := newTasks(stageID, domain.StatusInReview)
	calls := 0

	_, err := New(&fakeRepo{tasks: tasks}, fakeStages{calls: &calls}).Handle(context.Background(), Command{StageID: stageID, TaskIDs: ids, Status: domain.StatusInQA, Actor: dev})

	if !errors.Is(err, kernel.ErrTransitionForbidden) || calls != 0 {
		t.Fatalf("err = %v, consultas = %d", err, calls)
	}
}

func TestErrorKeepsKindOverREST(t *testing.T) {
	stageID := uuid.New()
	tasks, ids := newTasks(stageID, domain.StatusInReview)
	mux := http.NewServeMux()
	New(&fakeRepo{tasks: tasks}, fakeStages{team: team}).Register(mux)
	rec := httptest.NewRecorder()

	body := `{"task_ids":["` + ids[0].String() + `"],"status":"in_qa"}`
	req := httptest.NewRequest(http.MethodPut, "/api/stages/"+stageID.String()+"/tasks/status", strings.NewReader(body))
	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), dev)))

	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), ids[0].String()) {
		t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
