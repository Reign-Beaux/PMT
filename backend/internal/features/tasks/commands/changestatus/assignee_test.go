package changestatus

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

type failingStages struct{ fakeStages }

func (failingStages) Team(context.Context, uuid.UUID) (kernel.Team, error) {
	return kernel.Team{}, errors.New("no debió consultar el equipo")
}

func TestHandleTakeLeavesTaskToDev(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusReady)
	dev := kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}
	var saved domain.Task
	repo.onChange = func(task domain.Task) { saved = task }

	if _, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Status: domain.StatusInDev, Actor: dev}); err != nil {
		t.Fatal(err)
	}
	if saved.Assignee != "dev" {
		t.Errorf("assignee = %q", saved.Assignee)
	}
}

func TestHandleForbiddenProfileDoesNotQueryTeam(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusBacklog)
	cr := kernel.Actor{Name: "cr", Profile: kernel.ProfileCR}

	_, err := New(repo, failingStages{}).Handle(context.Background(), Command{ID: id, Status: domain.StatusReady, Actor: cr})

	if !errors.Is(err, kernel.ErrTransitionForbidden) || repo.updates != 0 {
		t.Fatalf("err = %v, updates = %d", err, repo.updates)
	}
}

func TestHandleRejectsOutsiders(t *testing.T) {
	tests := []struct {
		name  string
		from  domain.Status
		to    domain.Status
		actor kernel.Actor
		want  error
	}{
		{"TL ajeno", domain.StatusBacklog, domain.StatusReady, kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}, kernel.ErrNotStageLead},
		{"dev ajeno", domain.StatusReady, domain.StatusInDev, kernel.Actor{Name: "dev2", Profile: kernel.ProfileDevBackend}, domain.ErrNotStageMember},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)

			_, err := New(repo, fakeStages{}).Handle(context.Background(), Command{ID: id, Status: tt.to, Actor: tt.actor})

			if !errors.Is(err, tt.want) || repo.updates != 0 {
				t.Fatalf("err = %v, updates = %d", err, repo.updates)
			}
		})
	}
}
