package changestatus

import (
	"context"
	"errors"
	"testing"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

func TestHandleAppliesStatusRules(t *testing.T) {
	blocked := kernel.Conflict("regla")
	tl := actorFor("")

	tests := []struct {
		name   string
		from   domain.Status
		to     domain.Status
		stages fakeStages
		want   error
	}{
		{"avanzar permitido", domain.StatusBacklog, domain.StatusReady, fakeStages{cancel: blocked}, nil},
		{"avanzar rechazado", domain.StatusBacklog, domain.StatusReady, fakeStages{work: blocked}, blocked},
		{"cancelar permitido", domain.StatusReady, domain.StatusCancelled, fakeStages{work: blocked}, nil},
		{"cancelar rechazado", domain.StatusReady, domain.StatusCancelled, fakeStages{cancel: blocked}, blocked},
		{"reabrir permitido", domain.StatusCancelled, domain.StatusBacklog, fakeStages{work: blocked}, nil},
		{"reabrir rechazado", domain.StatusCancelled, domain.StatusBacklog, fakeStages{cancel: blocked}, blocked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)

			_, err := New(repo, tt.stages).Handle(context.Background(), Command{ID: id, Status: tt.to, Actor: tl, Reason: "motivo"})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if tt.want != nil && (repo.updates != 0 || repo.tasks[id].Status != tt.from) {
				t.Errorf("no debió guardar: %+v", repo.tasks[id])
			}
		})
	}
}

func TestHandleFailedRuleKeepsAssignee(t *testing.T) {
	blocked := kernel.Conflict("regla")
	repo, id := newRepoWith(t, domain.StatusReady)
	dev := kernel.Actor{Name: "dev", Profile: kernel.ProfileDevBackend}

	_, err := New(repo, fakeStages{work: blocked}).Handle(context.Background(), Command{ID: id, Status: domain.StatusInDev, Actor: dev})

	if !errors.Is(err, blocked) {
		t.Fatalf("err = %v", err)
	}
	if repo.tasks[id].Assignee != "" {
		t.Errorf("no debió tomarla: %+v", repo.tasks[id])
	}
}

func TestIsCancelOrReopen(t *testing.T) {
	tests := []struct {
		from, to domain.Status
		want     bool
	}{
		{domain.StatusInReview, domain.StatusCancelled, true},
		{domain.StatusCancelled, domain.StatusBacklog, true},
		{domain.StatusBacklog, domain.StatusReady, false},
		{domain.StatusPendingClose, domain.StatusDone, false},
	}
	for _, tt := range tests {
		if got := domain.IsCancelOrReopen(tt.from, tt.to); got != tt.want {
			t.Errorf("%s → %s = %v", tt.from, tt.to, got)
		}
	}
}
