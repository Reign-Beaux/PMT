package assign

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
	"pmt/internal/platform/rest"
)

type fakeRepo struct {
	task  domain.Task
	saved *domain.Task
}

func (r *fakeRepo) Get(context.Context, uuid.UUID) (domain.Task, error) {
	return r.task, nil
}

func (r *fakeRepo) UpdateAssignee(_ context.Context, task domain.Task) (domain.Task, error) {
	r.saved = &task
	return task, nil
}

type fakeStages struct{}

func (fakeStages) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	return kernel.Team{Lead: "tl", Members: []string{"dev", "dev2"}, ProjectID: project}, nil
}

type fakeAgents struct{ err error }

func (a fakeAgents) Profile(context.Context, string) (kernel.Profile, error) {
	return kernel.ProfileDevBackend, a.err
}

// fakeUsers son los usernames de usuarios activos, con el error que da cada uno
// según su rol en el proyecto: nil si es owner o editor.
type fakeUsers map[string]error

var (
	project    = uuid.MustParse("00000000-0000-0000-0000-000000000100")
	errNotWork = kernel.Conflict("el usuario es viewer del proyecto")
)

func (u fakeUsers) Worker(_ context.Context, projectID uuid.UUID, name string) (bool, error) {
	err, ok := u[name]
	if ok && projectID != project {
		return true, errors.New("consultó otro proyecto")
	}
	return ok, err
}

var (
	tl      = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	otherTL = kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}
)

func TestHandle(t *testing.T) {
	revoked := kernel.Conflict("revocado")
	tests := []struct {
		name   string
		status domain.Status
		actor  kernel.Actor
		agent  string
		agents fakeAgents
		users  fakeUsers
		want   error
	}{
		{"TL reasigna", domain.StatusInReview, tl, "dev2", fakeAgents{}, nil, nil},
		{"perfil incorrecto", domain.StatusInReview, kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}, "dev2", fakeAgents{}, nil, domain.ErrAssignForbidden},
		{"DP no asigna", domain.StatusInReview, kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}, "dev2", fakeAgents{}, nil, domain.ErrAssignForbidden},
		{"TL ajeno", domain.StatusInReview, otherTL, "dev2", fakeAgents{}, nil, kernel.ErrNotStageLead},
		{"fuera del equipo", domain.StatusInReview, tl, "otro", fakeAgents{}, nil, domain.ErrAssigneeNotMember},
		{"agente revocado", domain.StatusInReview, tl, "dev2", fakeAgents{err: revoked}, nil, revoked},
		{"usuario activo del equipo", domain.StatusInReview, tl, "dev2", fakeAgents{err: revoked}, fakeUsers{"dev2": nil}, nil},
		{"usuario del equipo viewer o no miembro", domain.StatusInReview, tl, "dev2", fakeAgents{}, fakeUsers{"dev2": errNotWork}, errNotWork},
		{"usuario activo fuera del equipo", domain.StatusInReview, tl, "otro", fakeAgents{}, fakeUsers{"otro": nil}, domain.ErrAssigneeNotMember},
		{"tarea libre", domain.StatusReady, tl, "dev2", fakeAgents{}, nil, domain.ErrTaskNotTaken},
		{"tarea cerrada", domain.StatusDone, tl, "dev2", fakeAgents{}, nil, domain.ErrTaskNotTaken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{task: domain.Task{ID: uuid.New(), Status: tt.status, Assignee: "dev"}}

			_, err := New(repo, fakeStages{}, tt.agents, tt.users).Handle(context.Background(), Command{ID: repo.task.ID, Agent: tt.agent, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && (repo.saved == nil || repo.saved.Assignee != tt.agent) {
				t.Fatalf("guardado = %+v", repo.saved)
			}
			if tt.want != nil && repo.saved != nil {
				t.Fatal("no debió guardar")
			}
		})
	}
}

func TestREST(t *testing.T) {
	repo := &fakeRepo{task: domain.Task{ID: uuid.New(), Status: domain.StatusInDev}}
	mux := http.NewServeMux()
	New(repo, fakeStages{}, fakeAgents{}, fakeUsers{}).Register(mux)
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/"+repo.task.ID.String()+"/assignee", strings.NewReader(`{"agent":"dev"}`))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req.WithContext(rest.WithActor(req.Context(), tl)))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"assignee":"dev"`) {
		t.Fatalf("status = %d, cuerpo = %s", rec.Code, rec.Body)
	}
}
