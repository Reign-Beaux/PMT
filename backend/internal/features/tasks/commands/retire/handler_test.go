package retire

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
)

type fakeStages struct {
	err error
}

func (s fakeStages) EnsureHistoryRule(context.Context, uuid.UUID) error {
	return s.err
}

type fakeRepo struct {
	existing uuid.UUID
	changes  []domain.Change
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	if id != r.existing {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return domain.Task{ID: id, Title: "Crear login"}, nil
}

func (r *fakeRepo) Retire(_ context.Context, change domain.Change) (domain.Change, error) {
	change.Version = 3
	r.changes = append(r.changes, change)
	return change, nil
}

func newHandler(repo Repository) *Handler {
	return New(repo, fakeStages{})
}

func TestHandleRetires(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id}

	result, err := newHandler(repo).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Reason: "No se pensó bien", RequestedBy: "PM"})

	if err != nil {
		t.Fatal(err)
	}
	if result.RetiredID != id || result.Version != 3 {
		t.Errorf("result = %+v", result)
	}
	if len(repo.changes) != 1 || repo.changes[0].Kind != domain.ChangeRetire || repo.changes[0].Title != "Crear login" {
		t.Errorf("cambios = %+v", repo.changes)
	}
}

func TestHandleRejectsInvalidInput(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name string
		cmd  Command
		want error
	}{
		{"sin motivo", Command{Actor: kernel.Owner, ID: id, RequestedBy: "PM"}, domain.ErrChangeReasonRequired},
		{"sin solicitante", Command{Actor: kernel.Owner, ID: id, Reason: "M"}, domain.ErrChangeRequesterRequired},
		{"no existe o ya retirada", Command{Actor: kernel.Owner, ID: uuid.New(), Reason: "M", RequestedBy: "PM"}, domain.ErrTaskNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{existing: id}

			_, err := newHandler(repo).Handle(context.Background(), tt.cmd)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if len(repo.changes) != 0 {
				t.Error("no debió retirar")
			}
		})
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	valid := `{"reason":"M","requested_by":"PM"}`
	tests := []struct {
		name   string
		id     string
		body   string
		status int
	}{
		{"válido", id.String(), valid, http.StatusOK},
		{"sin motivo", id.String(), `{"requested_by":"PM"}`, http.StatusBadRequest},
		{"id inválido", "abc", valid, http.StatusBadRequest},
		{"no existe", uuid.NewString(), valid, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			newHandler(&fakeRepo{existing: id}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/tasks/"+tt.id+"/retire", strings.NewReader(tt.body)))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandleRejectsStageInOwnPlanning(t *testing.T) {
	errOwnPlanning := kernel.Conflict("la etapa está en su propio planning")
	id := uuid.New()
	repo := &fakeRepo{existing: id}

	_, err := New(repo, fakeStages{err: errOwnPlanning}).Handle(context.Background(), Command{Actor: kernel.Owner, ID: id, Reason: "M", RequestedBy: "PM"})

	if !errors.Is(err, errOwnPlanning) || len(repo.changes) != 0 {
		t.Errorf("err = %v, cambios = %+v", err, repo.changes)
	}
}
