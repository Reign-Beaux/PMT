package assignlead

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	lead string
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	return domain.Stage{ID: id, ProjectID: project}, nil
}

func (r *fakeRepo) UpdateLead(_ context.Context, _ uuid.UUID, lead string) error {
	r.lead = lead
	return nil
}

type fakeAgents struct {
	profile kernel.Profile
	err     error
}

func (a fakeAgents) Profile(context.Context, string) (kernel.Profile, error) {
	return a.profile, a.err
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
	missing := kernel.NotFound("agente no encontrado")
	pp := kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	tests := []struct {
		name   string
		actor  kernel.Actor
		agents fakeAgents
		users  fakeUsers
		want   error
	}{
		{"PP asigna", pp, fakeAgents{profile: kernel.ProfileTL}, nil, nil},
		{"usuario asigna", kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}, fakeAgents{profile: kernel.ProfileTL}, nil, nil},
		{"TL no asigna", kernel.Actor{Profile: kernel.ProfileTL}, fakeAgents{profile: kernel.ProfileTL}, nil, domain.ErrAssignLeadForbidden},
		{"agente sin perfil TL", pp, fakeAgents{profile: kernel.ProfileDevBackend}, nil, domain.ErrAgentNotLead},
		{"agente inexistente, como el usuario", pp, fakeAgents{err: missing}, nil, missing},
		{"usuario owner o editor, sin revisar perfil", pp, fakeAgents{err: missing}, fakeUsers{"tl": nil}, nil},
		{"usuario viewer o no miembro", pp, fakeAgents{profile: kernel.ProfileTL}, fakeUsers{"tl": errNotWork}, errNotWork},
		{"usuario desactivado responde como hoy", pp, fakeAgents{err: missing}, fakeUsers{}, missing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			result, err := New(repo, tt.agents, tt.users).Handle(context.Background(), Command{ID: uuid.New(), Agent: "tl", Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && (repo.lead != "tl" || result.Lead != "tl") {
				t.Fatalf("lead = %q, resultado = %+v", repo.lead, result)
			}
			if tt.want != nil && repo.lead != "" {
				t.Fatal("no debió guardar")
			}
		})
	}
}
