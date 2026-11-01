package close

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

var (
	cr      = kernel.Actor{Name: "cr", Profile: kernel.ProfileCR}
	qa      = kernel.Actor{Name: "qa", Profile: kernel.ProfileQA}
	tl      = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	otherTL = kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}
	errGone = kernel.NotFound("tarea no encontrada")
)

type fakeRepo struct {
	finding domain.Finding
	closed  []domain.Finding
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Finding, error) {
	if id != r.finding.ID {
		return domain.Finding{}, domain.ErrFindingNotFound
	}
	return r.finding, nil
}

func (r *fakeRepo) Close(_ context.Context, f domain.Finding) (domain.Finding, error) {
	r.closed = append(r.closed, f)
	return f, nil
}

type fakeScope struct {
	gone  bool
	calls int
}

func (f *fakeScope) StageOf(context.Context, uuid.UUID) (uuid.UUID, error) {
	f.calls++
	if f.gone {
		return uuid.Nil, errGone
	}
	return uuid.New(), nil
}

func (f *fakeScope) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	return kernel.Team{Lead: tl.Name}, nil
}

func openFinding(kind domain.ItemKind) domain.Finding {
	return domain.Finding{ID: uuid.New(), ItemKind: kind, ItemID: uuid.New(), Status: domain.StatusOpen}
}

func TestHandleCloses(t *testing.T) {
	tests := []struct {
		name       string
		kind       domain.ItemKind
		resolution domain.Status
		actor      kernel.Actor
		calls      int
	}{
		{"CR corrige una tarea", domain.ItemTask, domain.StatusFixed, cr, 0},
		{"QA corrige un bug", domain.ItemBug, domain.StatusFixed, qa, 0},
		{"el TL de la etapa descarta", domain.ItemTask, domain.StatusDismissed, tl, 1},
		{"cualquier TL descarta en bugs", domain.ItemBug, domain.StatusDismissed, otherTL, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{finding: openFinding(tt.kind)}
			scope := &fakeScope{}

			result, err := New(repo, scope, scope).Handle(context.Background(), Command{ID: repo.finding.ID, Resolution: tt.resolution, Reason: "Motivo", Actor: tt.actor})

			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tt.resolution || result.ClosedBy != tt.actor.Name || result.CloseReason != "Motivo" || scope.calls != tt.calls {
				t.Errorf("result = %+v, consultas = %d", result, scope.calls)
			}
		})
	}
}

func TestHandleRejects(t *testing.T) {
	closed := openFinding(domain.ItemTask)
	closed.Status = domain.StatusFixed
	tests := []struct {
		name       string
		finding    domain.Finding
		resolution domain.Status
		actor      kernel.Actor
		reason     string
		gone       bool
		want       error
		calls      int
	}{
		{"TL ajeno descarta", openFinding(domain.ItemTask), domain.StatusDismissed, otherTL, "M", false, kernel.ErrNotStageLead, 1},
		{"CR descarta, sin consultar el ámbito", openFinding(domain.ItemTask), domain.StatusDismissed, cr, "M", false, domain.ErrDismissForbidden, 0},
		{"User descarta", openFinding(domain.ItemTask), domain.StatusDismissed, kernel.Owner, "M", false, domain.ErrDismissForbidden, 0},
		{"TL marca corregido", openFinding(domain.ItemTask), domain.StatusFixed, tl, "M", false, domain.ErrFixForbidden, 0},
		{"ya cerrado", closed, domain.StatusFixed, cr, "M", false, domain.ErrFindingClosed, 0},
		{"resolución desconocida", openFinding(domain.ItemTask), "open", cr, "M", false, domain.ErrResolutionInvalid, 0},
		{"sin motivo", openFinding(domain.ItemTask), domain.StatusFixed, cr, "  ", false, kernel.ErrReasonRequired, 0},
		{"TL ajeno sin motivo: primero el ámbito", openFinding(domain.ItemTask), domain.StatusDismissed, otherTL, "", false, kernel.ErrNotStageLead, 1},
		{"tarea que ya no existe", openFinding(domain.ItemTask), domain.StatusDismissed, tl, "M", true, errGone, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{finding: tt.finding}
			scope := &fakeScope{gone: tt.gone}

			_, err := New(repo, scope, scope).Handle(context.Background(), Command{ID: tt.finding.ID, Resolution: tt.resolution, Reason: tt.reason, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.closed) != 0 || scope.calls != tt.calls {
				t.Errorf("cerrados = %d, consultas = %d", len(repo.closed), scope.calls)
			}
		})
	}
}

func TestHandleRejectsMissingFinding(t *testing.T) {
	scope := &fakeScope{}

	_, err := New(&fakeRepo{}, scope, scope).Handle(context.Background(), Command{ID: uuid.New(), Resolution: domain.StatusFixed, Reason: "M", Actor: cr})

	if !errors.Is(err, domain.ErrFindingNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestREST(t *testing.T) {
	valid := `{"resolution":"fixed","reason":"M"}`
	finding := openFinding(domain.ItemBug)
	tests := []struct {
		name   string
		id     string
		body   string
		actor  kernel.Actor
		status int
	}{
		{"válido", finding.ID.String(), valid, cr, http.StatusOK},
		{"perfil sin permiso", finding.ID.String(), valid, tl, http.StatusForbidden},
		{"sin motivo", finding.ID.String(), `{"resolution":"fixed"}`, cr, http.StatusBadRequest},
		{"id inválido", "abc", valid, cr, http.StatusBadRequest},
		{"no existe", uuid.NewString(), valid, cr, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			scope := &fakeScope{}
			New(&fakeRepo{finding: finding}, scope, scope).Register(mux)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/findings/"+tt.id+"/close", strings.NewReader(tt.body))

			mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tt.actor)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestRESTConflictOnClosed(t *testing.T) {
	finding := openFinding(domain.ItemBug)
	finding.Status = domain.StatusDismissed
	mux := http.NewServeMux()
	scope := &fakeScope{}
	New(&fakeRepo{finding: finding}, scope, scope).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/findings/"+finding.ID.String()+"/close", strings.NewReader(`{"resolution":"fixed","reason":"M"}`))

	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), cr)))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d", rec.Code)
	}
}
