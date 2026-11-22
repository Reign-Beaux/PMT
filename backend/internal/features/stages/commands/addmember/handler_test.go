package addmember

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	added []string
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	return domain.Stage{ID: id, ProjectID: project, Lead: "tl", Members: []string{"dev"}}, nil
}

func (r *fakeRepo) AddMember(_ context.Context, _ uuid.UUID, agent, _ string) error {
	r.added = append(r.added, agent)
	return nil
}

var errAgentMissing = kernel.NotFound("agente no encontrado")

type fakeAgents struct{ profile kernel.Profile }

func (a fakeAgents) Profile(context.Context, string) (kernel.Profile, error) {
	if a.profile == "" {
		return "", errAgentMissing
	}
	return a.profile, nil
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

func TestHandle(t *testing.T) {
	tl := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	tests := []struct {
		name    string
		actor   kernel.Actor
		profile kernel.Profile
		users   fakeUsers
		want    error
	}{
		{"backend", tl, kernel.ProfileDevBackend, nil, nil},
		{"frontend", tl, kernel.ProfileDevFrontend, nil, nil},
		{"TL ajeno", kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}, kernel.ProfileDevBackend, nil, kernel.ErrNotStageLead},
		{"PP", kernel.Actor{Name: "tl", Profile: kernel.ProfilePP}, kernel.ProfileDevBackend, nil, kernel.ErrNotStageLead},
		{"usuario", kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}, kernel.ProfileDevBackend, nil, kernel.ErrNotStageLead},
		{"no es dev", tl, kernel.ProfileCR, nil, domain.ErrAgentNotDev},
		{"usuario owner o editor, sin revisar perfil", tl, "", fakeUsers{"dev2": nil}, nil},
		{"usuario viewer o no miembro", tl, kernel.ProfileDevBackend, fakeUsers{"dev2": errNotWork}, errNotWork},
		{"ni agente ni usuario activo", tl, "", fakeUsers{}, errAgentMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			result, err := New(repo, fakeAgents{tt.profile}, tt.users).Handle(context.Background(), Command{ID: uuid.New(), Agent: "dev2", Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && !slices.Equal(result.Members, []string{"dev", "dev2"}) {
				t.Fatalf("miembros = %v", result.Members)
			}
		})
	}
}
