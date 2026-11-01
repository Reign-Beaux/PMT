package changestatus

import (
	"context"
	"errors"
	"testing"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
)

func TestHandleRule15RequiresWorkingProject(t *testing.T) {
	notWorking := kernel.Conflict("el proyecto no está en working")
	tests := []struct {
		name     string
		projects fakeProjects
		want     error
		updates  int
	}{
		{"proyecto en working", fakeProjects{}, nil, 1},
		{"proyecto fuera de working", fakeProjects{err: notWorking}, notWorking, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, domain.StatusReported)

			_, err := New(repo, tt.projects).Handle(context.Background(), Command{ID: id, Status: domain.StatusConfirmed, Actor: actorFor(domain.StatusConfirmed)})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if repo.updates != tt.updates {
				t.Errorf("updates = %d", repo.updates)
			}
		})
	}
}

func TestHandleProfileFailsBeforeCheckingProject(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusReported)
	projects := fakeProjects{err: errors.New("no debió consultar el proyecto")}

	_, err := New(repo, projects).Handle(context.Background(), Command{ID: id, Status: domain.StatusConfirmed, Actor: kernel.Owner})

	if !errors.Is(err, kernel.ErrTransitionForbidden) {
		t.Fatalf("err = %v", err)
	}
}
