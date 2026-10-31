package changestatus

import (
	"context"
	"testing"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
)

func TestHandleRecordsStatusChange(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusBacklog)
	actor := kernel.Actor{Name: "agente", Profile: "PP"}

	_, err := New(repo).Handle(context.Background(), Command{ID: id, Status: domain.StatusReady, Reason: " motivo ", Actor: actor})

	if err != nil {
		t.Fatal(err)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("registros = %d", len(repo.recorded))
	}
	got := repo.recorded[0]
	if got.TaskID != id || got.From != domain.StatusBacklog || got.To != domain.StatusReady {
		t.Errorf("registro = %+v", got)
	}
	if got.ActorName != "agente" || got.ActorProfile != "PP" || got.Reason != "motivo" {
		t.Errorf("registro = %+v", got)
	}
}

func TestHandleRecordsNothingOnRejectedTransition(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusBacklog)

	_, err := New(repo).Handle(context.Background(), Command{ID: id, Status: domain.StatusBacklog, Actor: kernel.Anonymous})

	if err == nil {
		t.Fatal("se esperaba error")
	}
	if len(repo.recorded) != 0 {
		t.Errorf("registros = %d", len(repo.recorded))
	}
}
