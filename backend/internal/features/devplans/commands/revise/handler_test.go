package revise

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/devplans/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	dp       = kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}
	qa       = kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
	stageID  = uuid.New()
	bugID    = uuid.New()
	ownTask  = uuid.New()
	foreign  = uuid.New()
	errStage = kernel.NotFound("etapa no encontrada")
	errBug   = kernel.NotFound("bug no encontrado")
)

type fakeRepo struct {
	plan    domain.Plan
	gets    int
	revised int
}

func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Plan, error) {
	f.gets++
	if id != f.plan.ID {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	return f.plan, nil
}

func (f *fakeRepo) Revise(_ context.Context, _ uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error) {
	if err := apply(&f.plan); err != nil {
		return domain.Plan{}, err
	}
	f.revised++
	return f.plan, nil
}

type fakeTasks struct{}

func (fakeTasks) StageOf(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	if id == foreign {
		return uuid.New(), nil
	}
	return stageID, nil
}

type fakeChecker struct {
	exists bool
	err    error
}

func (f fakeChecker) Check(context.Context, uuid.UUID) error {
	if !f.exists {
		return f.err
	}
	return nil
}

func stagePlan() domain.Plan {
	return domain.Plan{ID: uuid.New(), StageID: &stageID, Latest: domain.Version{Version: 1}, TaskIDs: []uuid.UUID{ownTask}}
}

func bugPlan() domain.Plan {
	return domain.Plan{ID: uuid.New(), BugID: &bugID, Latest: domain.Version{Version: 1}}
}

func command(id uuid.UUID, tasks []uuid.UUID, reason string, actor kernel.Actor) Command {
	return Command{ID: id, TaskIDs: tasks, Scope: "S", TechnicalDecisions: "T", ScreenStates: "P", EdgeCases: "E", ErrorHandling: "F", AcceptanceCriteria: "A", Reason: reason, Actor: actor}
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name    string
		plan    domain.Plan
		tasks   []uuid.UUID
		reason  string
		actor   kernel.Actor
		exists  bool
		want    error
		version int
	}{
		{"plan de etapa", stagePlan(), []uuid.UUID{ownTask}, "motivo", dp, true, nil, 2},
		{"plan de bug", bugPlan(), nil, "motivo", dp, true, nil, 2},
		{"otro perfil", stagePlan(), []uuid.UUID{ownTask}, "motivo", qa, true, domain.ErrDPOnly, 1},
		{"sin motivo", stagePlan(), []uuid.UUID{ownTask}, " ", dp, true, domain.ErrVersionReasonRequired, 1},
		{"tarea de otra etapa", stagePlan(), []uuid.UUID{foreign}, "motivo", dp, true, domain.ErrTaskOutsideStage, 1},
		{"etapa eliminada", stagePlan(), []uuid.UUID{ownTask}, "motivo", dp, false, errStage, 1},
		{"bug inexistente", bugPlan(), nil, "motivo", dp, false, errBug, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{plan: tt.plan}
			h := New(repo, fakeTasks{}, fakeChecker{tt.exists, errStage}, fakeChecker{tt.exists, errBug})

			_, err := h.Handle(context.Background(), command(tt.plan.ID, tt.tasks, tt.reason, tt.actor))

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if repo.plan.Latest.Version != tt.version {
				t.Errorf("versión = %d", repo.plan.Latest.Version)
			}
			if tt.want != nil && repo.revised != 0 {
				t.Error("guardó al rechazarse")
			}
		})
	}
}

func TestProfileFailsBeforeReading(t *testing.T) {
	repo := &fakeRepo{plan: stagePlan()}
	h := New(repo, fakeTasks{}, fakeChecker{}, fakeChecker{})

	_, err := h.Handle(context.Background(), command(uuid.New(), nil, "motivo", qa))

	if !errors.Is(err, domain.ErrDPOnly) || repo.gets != 0 {
		t.Fatalf("err = %v, lecturas = %d", err, repo.gets)
	}
}

func TestREST(t *testing.T) {
	plan := bugPlan()
	body := `{"scope":"S","technical_decisions":"T","screen_states":"P","edge_cases":"E","error_handling":"F","acceptance_criteria":"A","reason":"motivo"}`
	tests := []struct {
		name   string
		id     string
		actor  kernel.Actor
		status int
	}{
		{"válido", plan.ID.String(), dp, http.StatusOK},
		{"perfil sin permiso", plan.ID.String(), qa, http.StatusForbidden},
		{"inexistente", uuid.NewString(), dp, http.StatusNotFound},
		{"id inválido", "abc", dp, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{plan: plan}, fakeTasks{}, fakeChecker{exists: true}, fakeChecker{exists: true}).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/dev-plans/"+tt.id+"/revise", strings.NewReader(body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tt.actor)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
