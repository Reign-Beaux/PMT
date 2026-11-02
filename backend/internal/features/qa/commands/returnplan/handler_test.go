package returnplan

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
	tl      = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	otherTL = kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}
	qa      = kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
)

type fakeRepo struct{ plan domain.Plan }

func (f *fakeRepo) Get(context.Context, uuid.UUID) (domain.Plan, error) { return f.plan, nil }

func (f *fakeRepo) Decide(_ context.Context, _ uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error) {
	if err := apply(&f.plan); err != nil {
		return domain.Plan{}, err
	}
	return f.plan, nil
}

type fakeStages struct{ calls int }

func (f *fakeStages) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	f.calls++
	return kernel.Team{Lead: tl.Name}, nil
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		notes  string
		actor  kernel.Actor
		want   error
		calls  int
	}{
		{"válido", domain.StatusSubmitted, "Falta X", tl, nil, 1},
		{"TL ajeno sin correcciones: primero el ámbito", domain.StatusSubmitted, "", otherTL, kernel.ErrNotStageLead, 1},
		{"sin correcciones", domain.StatusSubmitted, "  ", tl, domain.ErrReviewNotesRequired, 1},
		{"perfil falla antes de consultar", domain.StatusSubmitted, "X", qa, domain.ErrTLOnly, 0},
		{"aprobado", domain.StatusApproved, "X", tl, domain.ErrPlanNotSubmitted, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{plan: domain.Plan{ID: uuid.New(), Status: tt.status}}
			stages := &fakeStages{}
			result, err := New(repo, stages).Handle(context.Background(), Command{ID: repo.plan.ID, ReviewNotes: tt.notes, Actor: tt.actor})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if stages.calls != tt.calls {
				t.Errorf("consultas = %d", stages.calls)
			}
			if err == nil && (result.Status != domain.StatusReturned || result.ReviewNotes != tt.notes) {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"válido", `{"review_notes":"Falta X"}`, http.StatusOK},
		{"sin correcciones", `{}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			repo := &fakeRepo{plan: domain.Plan{ID: uuid.New(), Status: domain.StatusSubmitted}}
			New(repo, &fakeStages{}).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/test-plans/"+repo.plan.ID.String()+"/return", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tl)))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
