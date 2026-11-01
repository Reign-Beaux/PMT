package changestatus

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

func TestHandleRecordsStatusChange(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusPlanning)
	actor := kernel.Actor{Name: "agente", Profile: "PP"}

	_, err := New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusStandBy, Reason: " motivo ", Actor: actor})

	if err != nil {
		t.Fatal(err)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("registros = %d", len(repo.recorded))
	}
	got := repo.recorded[0]
	if got.StageID != id || got.From != domain.StatusPlanning || got.To != domain.StatusStandBy {
		t.Errorf("registro = %+v", got)
	}
	if got.ActorName != "agente" || got.ActorProfile != "PP" || got.Reason != "motivo" {
		t.Errorf("registro = %+v", got)
	}
}

func TestHandleRecordsNothingOnRejectedTransition(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusPlanning)

	_, err := New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: id, Status: domain.StatusPlanning, Actor: kernel.Owner})

	if err == nil {
		t.Fatal("se esperaba error")
	}
	if len(repo.recorded) != 0 {
		t.Errorf("registros = %d", len(repo.recorded))
	}
}

func TestHandleRecordsStatusChangeLeavingOwnPlanning(t *testing.T) {
	s, err := domain.NewOwnPlanningStage(uuid.New(), "Pagos", "Inicial", "Petición", "PM")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{stages: map[uuid.UUID]domain.Stage{s.ID: s}}

	_, err = New(repo, &fakeTasks{}, workingProject).Handle(context.Background(), Command{ID: s.ID, Status: domain.StatusStandBy, Reason: "listo", Actor: actorFor(domain.StatusStandBy)})

	if err != nil {
		t.Fatal(err)
	}
	if len(repo.recorded) != 1 || repo.recorded[0].Reason != "listo" || repo.recorded[0].From != domain.StatusPlanning {
		t.Errorf("registros = %+v", repo.recorded)
	}
}
