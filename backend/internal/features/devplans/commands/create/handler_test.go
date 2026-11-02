package create

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
	tl       = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	stageID  = uuid.New()
	bugID    = uuid.New()
	ownTask  = uuid.New()
	foreign  = uuid.New()
	errGone  = kernel.NotFound("tarea no encontrada")
	errStage = kernel.NotFound("etapa no encontrada")
	errBug   = kernel.NotFound("bug no encontrado")
)

type fakeRepo struct{ created []domain.Plan }

func (f *fakeRepo) Create(_ context.Context, p domain.Plan) (domain.Plan, error) {
	f.created = append(f.created, p)
	return p, nil
}

type fakeTasks struct{ calls *int }

func (f fakeTasks) StageOf(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	*f.calls++
	switch id {
	case ownTask:
		return stageID, nil
	case foreign:
		return uuid.New(), nil
	}
	return uuid.Nil, errGone
}

type fakeChecker struct {
	calls *int
	known uuid.UUID
	err   error
}

func (f fakeChecker) Check(_ context.Context, id uuid.UUID) error {
	*f.calls++
	if id != f.known {
		return f.err
	}
	return nil
}

func newHandler(repo *fakeRepo, calls *int) *Handler {
	return New(repo, fakeTasks{calls}, fakeChecker{calls, stageID, errStage}, fakeChecker{calls, bugID, errBug})
}

func command(stage, bug *uuid.UUID, tasks []uuid.UUID, actor kernel.Actor) Command {
	return Command{StageID: stage, BugID: bug, TaskIDs: tasks, Scope: "S", TechnicalDecisions: "T", ScreenStates: "P", EdgeCases: "E", ErrorHandling: "F", AcceptanceCriteria: "A", Actor: actor}
}

func TestHandle(t *testing.T) {
	missing := uuid.New()
	tests := []struct {
		name  string
		stage *uuid.UUID
		bug   *uuid.UUID
		tasks []uuid.UUID
		actor kernel.Actor
		want  error
		calls int
	}{
		{"por etapa", &stageID, nil, []uuid.UUID{ownTask}, dp, nil, 2},
		{"por bug", nil, &bugID, nil, dp, nil, 1},
		{"perfil falla antes de consultar", &stageID, nil, []uuid.UUID{ownTask}, tl, domain.ErrDPOnly, 0},
		{"los dos modos", &stageID, &bugID, nil, dp, domain.ErrModeInvalid, 0},
		{"etapa inexistente", &missing, nil, []uuid.UUID{ownTask}, dp, errStage, 1},
		{"bug inexistente", nil, &missing, nil, dp, errBug, 1},
		{"tarea de otra etapa", &stageID, nil, []uuid.UUID{foreign}, dp, domain.ErrTaskOutsideStage, 2},
		{"tarea inexistente", &stageID, nil, []uuid.UUID{missing}, dp, errGone, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, calls := &fakeRepo{}, 0
			result, err := newHandler(repo, &calls).Handle(context.Background(), command(tt.stage, tt.bug, tt.tasks, tt.actor))
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if calls != tt.calls {
				t.Errorf("consultas = %d", calls)
			}
			if tt.want != nil {
				if len(repo.created) != 0 {
					t.Error("guardó al rechazarse")
				}
				return
			}
			if result.Version != 1 {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestREST(t *testing.T) {
	fields := `"scope":"S","technical_decisions":"T","screen_states":"P","edge_cases":"E","error_handling":"F","acceptance_criteria":"A"`
	byStage := `{"stage_id":"` + stageID.String() + `","task_ids":["` + ownTask.String() + `"],` + fields + `}`
	byBug := `{"bug_id":"` + bugID.String() + `",` + fields + `}`
	tests := []struct {
		name   string
		body   string
		actor  kernel.Actor
		status int
	}{
		{"por etapa", byStage, dp, http.StatusCreated},
		{"por bug", byBug, dp, http.StatusCreated},
		{"perfil sin permiso", byStage, tl, http.StatusForbidden},
		{"sin modo", `{` + fields + `}`, dp, http.StatusBadRequest},
		{"sin contenido", `{"bug_id":"` + bugID.String() + `"}`, dp, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			calls := 0
			newHandler(&fakeRepo{}, &calls).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/dev-plans", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tt.actor)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
