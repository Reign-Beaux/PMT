package removemember

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

type fakeRepo struct {
	removed bool
}

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (domain.Stage, error) {
	return domain.Stage{ID: id, Lead: "tl", Members: []string{"dev", "dev2"}}, nil
}

func (r *fakeRepo) RemoveMember(context.Context, uuid.UUID, string) error {
	r.removed = true
	return nil
}

type fakeTasks struct{ open bool }

func (t fakeTasks) HasOpenAssigned(context.Context, uuid.UUID, string) (bool, error) {
	return t.open, nil
}

func TestHandle(t *testing.T) {
	tl := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	tests := []struct {
		name  string
		actor kernel.Actor
		open  bool
		want  error
	}{
		{"quita", tl, false, nil},
		{"con tareas abiertas", tl, true, domain.ErrMemberHasTasks},
		{"TL ajeno", kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}, false, kernel.ErrNotStageLead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}

			result, err := New(repo, fakeTasks{tt.open}).Handle(context.Background(), Command{ID: uuid.New(), Agent: "dev", Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if repo.removed != (tt.want == nil) {
				t.Fatalf("removed = %v", repo.removed)
			}
			if tt.want == nil && len(result.Members) != 1 {
				t.Fatalf("miembros = %v", result.Members)
			}
		})
	}
}
