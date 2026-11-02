package revise

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	qap     = kernel.Actor{Name: "qap", Profile: kernel.ProfileQAP}
	tl      = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	stageID = uuid.New()
	ownTask = uuid.New()
	foreign = uuid.New()
)

type fakeRepo struct {
	plan    domain.Plan
	revised int
}

func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Plan, error) {
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

type fakeTasks struct{ calls int }

func (f *fakeTasks) StageOf(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	f.calls++
	if id == foreign {
		return uuid.New(), nil
	}
	return stageID, nil
}

func newPlan(status domain.Status) domain.Plan {
	return domain.Plan{ID: uuid.New(), StageID: stageID, Status: status, Latest: domain.Version{Version: 1}}
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		task   uuid.UUID
		actor  kernel.Actor
		want   error
		calls  int
	}{
		{"desde returned", domain.StatusReturned, ownTask, qap, nil, 1},
		{"desde approved", domain.StatusApproved, ownTask, qap, nil, 1},
		{"enviado", domain.StatusSubmitted, ownTask, qap, domain.ErrPlanSubmitted, 0},
		{"perfil falla antes de consultar", domain.StatusDraft, ownTask, tl, domain.ErrQAPOnly, 0},
		{"tarea de otra etapa", domain.StatusDraft, foreign, qap, domain.ErrTaskOutsideStage, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tasks := &fakeRepo{plan: newPlan(tt.status)}, &fakeTasks{}
			result, err := New(repo, tasks).Handle(context.Background(), Command{ID: repo.plan.ID, Reason: "M", TaskIDs: []uuid.UUID{tt.task}, Actor: tt.actor})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if tasks.calls != tt.calls {
				t.Errorf("consultas = %d", tasks.calls)
			}
			if tt.want != nil {
				if repo.revised != 0 {
					t.Error("guardó al rechazarse")
				}
				return
			}
			if result.Status != domain.StatusDraft || result.Version != 2 {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestREST(t *testing.T) {
	valid := `{"reason":"M","task_ids":["` + ownTask.String() + `"]}`
	tests := []struct {
		name   string
		status domain.Status
		body   string
		code   int
	}{
		{"válido", domain.StatusDraft, valid, http.StatusOK},
		{"enviado", domain.StatusSubmitted, valid, http.StatusConflict},
		{"sin motivo", domain.StatusDraft, `{"task_ids":["` + ownTask.String() + `"]}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			repo := &fakeRepo{plan: newPlan(tt.status)}
			New(repo, &fakeTasks{}).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/test-plans/"+repo.plan.ID.String()+"/revise", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), qap)))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
