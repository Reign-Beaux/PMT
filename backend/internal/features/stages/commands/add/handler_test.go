package add

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	saved   []domain.Stage
	changes []domain.Change
	err     error
}

func (r *fakeRepo) Add(_ context.Context, stage domain.Stage, change domain.Change) (domain.Stage, domain.Change, error) {
	if r.err != nil {
		return domain.Stage{}, domain.Change{}, r.err
	}
	change.Version = 1
	r.saved = append(r.saved, stage)
	r.changes = append(r.changes, change)
	return stage, change, nil
}

func (r *fakeRepo) Create(_ context.Context, stage domain.Stage) (domain.Stage, error) {
	if r.err != nil {
		return domain.Stage{}, r.err
	}
	r.saved = append(r.saved, stage)
	return stage, nil
}

type fakeProjects struct {
	working bool
	err     error
}

func (p fakeProjects) IsWorking(context.Context, uuid.UUID) (bool, error) {
	return p.working, p.err
}

func newHandler(repo Repository) *Handler {
	return New(repo, fakeProjects{})
}

func TestHandleAdds(t *testing.T) {
	repo := &fakeRepo{}
	parent := uuid.New()

	result, err := newHandler(repo).Handle(context.Background(), Command{
		Actor:     kernel.Owner,
		ProjectID: parent, Name: "Nueva", Reason: "Faltaba", RequestedBy: "PM",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Nueva" || result.ProjectID != parent || result.Version != 1 {
		t.Errorf("result = %+v", result)
	}
	if len(repo.changes) != 1 || repo.changes[0].Kind != domain.ChangeAdd || repo.changes[0].Reason != "Faltaba" {
		t.Errorf("cambios = %+v", repo.changes)
	}
}

func TestHandleRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want error
	}{
		{"sin name", Command{Actor: kernel.Owner, Reason: "M", RequestedBy: "PM"}, domain.ErrStageNameRequired},
		{"sin motivo", Command{Actor: kernel.Owner, Name: "A", RequestedBy: "PM"}, domain.ErrChangeReasonRequired},
		{"sin solicitante", Command{Actor: kernel.Owner, Name: "A", Reason: "M"}, domain.ErrChangeRequesterRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			_, err := newHandler(repo).Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar")
			}
		})
	}
}

func TestREST(t *testing.T) {
	path := "/api/projects/" + uuid.NewString() + "/stages/add"
	valid := `{"name":"A","reason":"M","requested_by":"PM"}`
	tests := []struct {
		name   string
		path   string
		body   string
		err    error
		status int
	}{
		{"válida", path, valid, nil, http.StatusCreated},
		{"sin motivo", path, `{"name":"A","requested_by":"PM"}`, nil, http.StatusBadRequest},
		{"id inválido", "/api/projects/abc/stages/add", valid, nil, http.StatusBadRequest},
		{"campo desconocido", path, `{"name":"A","version":2}`, nil, http.StatusBadRequest},
		{"padre inexistente", path, valid, domain.ErrProjectNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			newHandler(&fakeRepo{err: tt.err}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandleBornsInOwnPlanningWhenProjectIsWorking(t *testing.T) {
	repo := &fakeRepo{}

	result, err := New(repo, fakeProjects{working: true}).Handle(context.Background(), Command{
		Actor:     kernel.Owner,
		ProjectID: uuid.New(), Name: "Pagos", Reason: " Petición ", RequestedBy: "PM",
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 0 || result.AddReason != "Petición" || result.AddRequestedBy != "PM" {
		t.Errorf("result = %+v", result)
	}
	if len(repo.changes) != 0 {
		t.Errorf("no debió escribir registros: %+v", repo.changes)
	}
}

func TestHandleRequiresReasonInOwnPlanning(t *testing.T) {
	repo := &fakeRepo{}

	_, err := New(repo, fakeProjects{working: true}).Handle(context.Background(), Command{
		Actor:     kernel.Owner,
		ProjectID: uuid.New(), Name: "Pagos", RequestedBy: "PM",
	})

	if !errors.Is(err, domain.ErrChangeReasonRequired) || len(repo.saved) != 0 {
		t.Errorf("err = %v, guardadas = %+v", err, repo.saved)
	}
}

func TestHandleReturnsProjectError(t *testing.T) {
	_, err := New(&fakeRepo{}, fakeProjects{err: domain.ErrProjectNotFound}).Handle(context.Background(), Command{
		Actor:     kernel.Owner,
		ProjectID: uuid.New(), Name: "Pagos", Reason: "M", RequestedBy: "PM",
	})

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("err = %v", err)
	}
}
