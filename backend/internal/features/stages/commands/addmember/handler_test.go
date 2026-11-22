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
	return domain.Stage{ID: id, Lead: "tl", Members: []string{"dev"}}, nil
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

// fakeUsers son los usernames de usuarios activos.
type fakeUsers map[string]bool

func (u fakeUsers) Active(_ context.Context, name string) (bool, error) {
	return u[name], nil
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
		{"usuario activo, sin revisar perfil", tl, "", fakeUsers{"dev2": true}, nil},
		{"ni agente ni usuario activo", tl, "", fakeUsers{"dev2": false}, errAgentMissing},
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
