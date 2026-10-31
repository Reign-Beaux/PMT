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

type fakeAgents struct{ profile kernel.Profile }

func (a fakeAgents) Profile(context.Context, string) (kernel.Profile, error) {
	return a.profile, nil
}

func TestHandle(t *testing.T) {
	tl := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	tests := []struct {
		name    string
		actor   kernel.Actor
		profile kernel.Profile
		want    error
	}{
		{"backend", tl, kernel.ProfileDevBackend, nil},
		{"frontend", tl, kernel.ProfileDevFrontend, nil},
		{"TL ajeno", kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}, kernel.ProfileDevBackend, kernel.ErrNotStageLead},
		{"PP", kernel.Actor{Name: "tl", Profile: kernel.ProfilePP}, kernel.ProfileDevBackend, kernel.ErrNotStageLead},
		{"usuario", kernel.Owner, kernel.ProfileDevBackend, kernel.ErrNotStageLead},
		{"no es dev", tl, kernel.ProfileCR, domain.ErrAgentNotDev},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			result, err := New(repo, fakeAgents{tt.profile}).Handle(context.Background(), Command{ID: uuid.New(), Agent: "dev2", Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if tt.want == nil && !slices.Equal(result.Members, []string{"dev", "dev2"}) {
				t.Fatalf("miembros = %v", result.Members)
			}
		})
	}
}
