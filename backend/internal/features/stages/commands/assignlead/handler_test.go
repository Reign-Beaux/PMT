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
	return domain.Stage{ID: id}, nil
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

func TestHandle(t *testing.T) {
	missing := kernel.NotFound("agente no encontrado")
	pp := kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}
	tests := []struct {
		name   string
		actor  kernel.Actor
		agents fakeAgents
		want   error
	}{
		{"PP asigna", pp, fakeAgents{profile: kernel.ProfileTL}, nil},
		{"usuario asigna", kernel.Owner, fakeAgents{profile: kernel.ProfileTL}, nil},
		{"TL no asigna", kernel.Actor{Profile: kernel.ProfileTL}, fakeAgents{profile: kernel.ProfileTL}, domain.ErrAssignLeadForbidden},
		{"agente sin perfil TL", pp, fakeAgents{profile: kernel.ProfileDevBackend}, domain.ErrAgentNotLead},
		{"agente inexistente, como el usuario", pp, fakeAgents{err: missing}, missing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			result, err := New(repo, tt.agents).Handle(context.Background(), Command{ID: uuid.New(), Agent: "tl", Actor: tt.actor})

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
