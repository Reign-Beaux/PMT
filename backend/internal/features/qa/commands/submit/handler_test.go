package submit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/qa/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	qap = kernel.Actor{Name: "qap", Profile: kernel.ProfileQAP}
	tl  = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
)

type fakeRepo struct{ plan domain.Plan }

func (f *fakeRepo) Decide(_ context.Context, id uuid.UUID, apply func(*domain.Plan) error) (domain.Plan, error) {
	if id != f.plan.ID {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	if err := apply(&f.plan); err != nil {
		return domain.Plan{}, err
	}
	return f.plan, nil
}

func newPlan(status domain.Status) domain.Plan {
	return domain.Plan{ID: uuid.New(), Status: status, Latest: domain.Version{Version: 1, HappyPath: "H", ErrorCases: "E"}}
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		actor  kernel.Actor
		want   error
	}{
		{"válido", domain.StatusDraft, qap, nil},
		{"perfil sin permiso", domain.StatusDraft, tl, domain.ErrQAPOnly},
		{"devuelto", domain.StatusReturned, qap, domain.ErrReturnedNeedsVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{plan: newPlan(tt.status)}
			result, err := New(repo).Handle(context.Background(), Command{ID: repo.plan.ID, Actor: tt.actor})
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if err == nil && result.Status != domain.StatusSubmitted {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestREST(t *testing.T) {
	tests := []struct {
		name  string
		actor kernel.Actor
		id    func(domain.Plan) string
		code  int
	}{
		{"válido", qap, func(p domain.Plan) string { return p.ID.String() }, http.StatusOK},
		{"perfil sin permiso", tl, func(p domain.Plan) string { return p.ID.String() }, http.StatusForbidden},
		{"no existe", qap, func(domain.Plan) string { return uuid.NewString() }, http.StatusNotFound},
		{"id inválido", qap, func(domain.Plan) string { return "abc" }, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			repo := &fakeRepo{plan: newPlan(domain.StatusDraft)}
			New(repo).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/test-plans/"+tt.id(repo.plan)+"/submit", nil)

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tt.actor)))

			if rec.Code != tt.code {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
