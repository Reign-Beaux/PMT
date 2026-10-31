package changestatus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/bugs/domain"
)

type fakeRepo struct {
	bugs    map[uuid.UUID]domain.Bug
	updates int
}

func newRepoWith(t *testing.T, status domain.Status) (*fakeRepo, uuid.UUID) {
	t.Helper()
	bug, err := domain.NewBug(uuid.New(), "Falla", "Detalle", "Paso 1", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	bug.Status = status
	return &fakeRepo{bugs: map[uuid.UUID]domain.Bug{bug.ID: bug}}, bug.ID
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Bug, error) {
	bug, ok := r.bugs[id]
	if !ok {
		return domain.Bug{}, domain.ErrBugNotFound
	}
	return bug, nil
}

func (r *fakeRepo) UpdateStatus(_ context.Context, bug domain.Bug) (domain.Bug, error) {
	r.updates++
	row := r.bugs[bug.ID]
	row.Status = bug.Status
	r.bugs[bug.ID] = row
	return row, nil
}

func TestHandleChangesStatus(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusReported)

	result, err := New(repo).Handle(context.Background(), Command{ID: id, Status: domain.StatusConfirmed})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.StatusConfirmed || repo.bugs[id].Status != domain.StatusConfirmed {
		t.Errorf("result = %+v", result)
	}
	if result.Title != "Falla" || result.StepsToReproduce != "Paso 1" || result.ProductVersion != "1.0.0" {
		t.Errorf("debe devolver el bug completo: %+v", result)
	}
}

func TestHandleRejectedStatusIsNotSaved(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		want   error
	}{
		{"transición no permitida", domain.StatusDone, domain.ErrBugStatusTransition},
		{"mismo status", domain.StatusReported, domain.ErrBugStatusTransition},
		{"status desconocido", "archived", domain.ErrBugStatusInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, domain.StatusReported)

			_, err := New(repo).Handle(context.Background(), Command{ID: id, Status: tt.status})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if repo.updates != 0 || repo.bugs[id].Status != domain.StatusReported {
				t.Errorf("no debió guardar: %+v", repo.bugs[id])
			}
		})
	}
}

func TestHandleNotFound(t *testing.T) {
	repo, _ := newRepoWith(t, domain.StatusReported)

	_, err := New(repo).Handle(context.Background(), Command{ID: uuid.New(), Status: domain.StatusConfirmed})

	if !errors.Is(err, domain.ErrBugNotFound) {
		t.Fatalf("err = %v", err)
	}
	if repo.updates != 0 {
		t.Error("no debió guardar el status")
	}
}

func TestRest(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		path func(uuid.UUID) string
		body string
		want int
	}{
		{"cambia", domain.StatusReported, bugPath, `{"status":"confirmed"}`, http.StatusOK},
		{"id inválido", domain.StatusReported, func(uuid.UUID) string { return "/api/bugs/abc/status" }, `{"status":"confirmed"}`, http.StatusBadRequest},
		{"campo desconocido", domain.StatusReported, bugPath, `{"status":"confirmed","x":1}`, http.StatusBadRequest},
		{"status desconocido", domain.StatusReported, bugPath, `{"status":"archived"}`, http.StatusBadRequest},
		{"transición no permitida", domain.StatusDone, bugPath, `{"status":"reported"}`, http.StatusConflict},
		{"inexistente", domain.StatusReported, func(uuid.UUID) string { return "/api/bugs/" + uuid.NewString() + "/status" }, `{"status":"confirmed"}`, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)
			mux := http.NewServeMux()
			New(repo).Register(mux)

			req := httptest.NewRequest(http.MethodPut, tt.path(id), strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("código = %d, cuerpo = %s", rec.Code, rec.Body)
			}
			if tt.want != http.StatusOK {
				return
			}
			var result Result
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.ID != id || result.Status != domain.StatusConfirmed || result.Title != "Falla" {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func bugPath(id uuid.UUID) string {
	return "/api/bugs/" + id.String() + "/status"
}
