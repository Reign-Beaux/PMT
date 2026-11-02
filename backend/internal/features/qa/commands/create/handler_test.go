package create

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
	qap      = kernel.Actor{Name: "qap", Profile: kernel.ProfileQAP}
	tl       = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	stageID  = uuid.New()
	ownTask  = uuid.New()
	foreign  = uuid.New()
	errGone  = kernel.NotFound("tarea no encontrada")
	errStage = kernel.NotFound("etapa no encontrada")
)

type fakeRepo struct{ created []domain.Plan }

func (f *fakeRepo) Create(_ context.Context, p domain.Plan) (domain.Plan, error) {
	f.created = append(f.created, p)
	return p, nil
}

type fakeScope struct{ calls int }

func (f *fakeScope) StageOf(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	f.calls++
	switch id {
	case ownTask:
		return stageID, nil
	case foreign:
		return uuid.New(), nil
	}
	return uuid.Nil, errGone
}

func (f *fakeScope) Check(_ context.Context, id uuid.UUID) error {
	f.calls++
	if id != stageID {
		return errStage
	}
	return nil
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name  string
		stage uuid.UUID
		task  uuid.UUID
		actor kernel.Actor
		want  error
		calls int
	}{
		{"válido", stageID, ownTask, qap, nil, 2},
		{"perfil falla antes de consultar", stageID, ownTask, tl, domain.ErrQAPOnly, 0},
		{"etapa inexistente", uuid.New(), ownTask, qap, errStage, 1},
		{"tarea de otra etapa", stageID, foreign, qap, domain.ErrTaskOutsideStage, 2},
		{"tarea inexistente", stageID, uuid.New(), qap, errGone, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, scope := &fakeRepo{}, &fakeScope{}
			result, err := New(repo, scope, scope).Handle(context.Background(), Command{StageID: tt.stage, Title: "Plan", TaskIDs: []uuid.UUID{tt.task}, Actor: tt.actor})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if scope.calls != tt.calls {
				t.Errorf("consultas = %d", scope.calls)
			}
			if tt.want != nil {
				if len(repo.created) != 0 {
					t.Error("guardó al rechazarse")
				}
				return
			}
			if result.Status != domain.StatusDraft || result.Version != 1 {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestREST(t *testing.T) {
	valid := `{"title":"Plan","task_ids":["` + ownTask.String() + `"]}`
	tests := []struct {
		name   string
		body   string
		actor  kernel.Actor
		status int
	}{
		{"válido", valid, qap, http.StatusCreated},
		{"perfil sin permiso", valid, tl, http.StatusForbidden},
		{"sin tareas", `{"title":"Plan"}`, qap, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			scope := &fakeScope{}
			New(&fakeRepo{}, scope, scope).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/stages/"+stageID.String()+"/test-plans", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tt.actor)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
