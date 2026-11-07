package delete

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	existing uuid.UUID
	status   domain.Status
	err      error
	deleted  bool
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	if id != r.existing {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return domain.Project{ID: id, Status: r.status}, nil
}

func (r *fakeRepo) Delete(context.Context, uuid.UUID) error {
	if r.err != nil {
		return r.err
	}
	r.deleted = true
	return nil
}

func TestHandleDeletesInPlanning(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id, status: domain.StatusPlanning}

	if err := New(repo).Handle(context.Background(), id, kernel.Owner); err != nil {
		t.Fatal(err)
	}
	if !repo.deleted {
		t.Error("debió eliminar el proyecto")
	}
}

func TestHandleRejectsOutsidePlanning(t *testing.T) {
	statuses := []domain.Status{
		domain.StatusStandBy, domain.StatusInDev, domain.StatusWorking, domain.StatusStopped, domain.StatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			id := uuid.New()
			repo := &fakeRepo{existing: id, status: status}

			err := New(repo).Handle(context.Background(), id, kernel.Owner)

			if !errors.Is(err, domain.ErrProjectNotPlanning) {
				t.Fatalf("err = %v", err)
			}
			if repo.deleted {
				t.Error("no debió eliminar el proyecto")
			}
		})
	}
}

func TestREST(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name   string
		id     string
		state  domain.Status
		err    error
		status int
	}{
		{"existe", id.String(), domain.StatusPlanning, nil, http.StatusNoContent},
		{"id inválido", "abc", domain.StatusPlanning, nil, http.StatusBadRequest},
		{"no existe", uuid.NewString(), domain.StatusPlanning, nil, http.StatusNotFound},
		{"con etapas", id.String(), domain.StatusPlanning, domain.ErrProjectHasStages, http.StatusConflict},
		{"fuera de planning", id.String(), domain.StatusStandBy, nil, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			New(&fakeRepo{existing: id, status: tt.state, err: tt.err}).Register(mux)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/projects/"+tt.id, nil))

			if rec.Code != tt.status {
				t.Errorf("status = %d, cuerpo = %s", rec.Code, rec.Body)
			}
		})
	}
}
