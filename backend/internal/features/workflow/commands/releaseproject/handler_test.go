package releaseproject

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	"pmt/internal/features/workflow/domain"
	"pmt/internal/kernel"
)

var pp = kernel.Actor{Name: "pp", Profile: kernel.ProfilePP}

type fakeRepo struct {
	project projects.Project
	stages  []stages.Stage
	plan    *domain.ReleasePlan
}

func (r *fakeRepo) Project(context.Context, uuid.UUID) (projects.Project, error) {
	return r.project, nil
}

func (r *fakeRepo) Stages(context.Context, uuid.UUID) ([]stages.Stage, error) {
	return r.stages, nil
}

func (r *fakeRepo) Release(_ context.Context, plan domain.ReleasePlan) error {
	r.plan = &plan
	return nil
}

type fakeTasks struct {
	failFor uuid.UUID
	err     error
}

func (f fakeTasks) EnsureReleasable(_ context.Context, id uuid.UUID) error {
	if id == f.failFor {
		return f.err
	}
	return nil
}

func stage(status stages.Status) stages.Stage {
	return stages.Stage{ID: uuid.New(), Name: "Etapa", Status: status}
}

func TestHandleMovesOnlyPlanningStages(t *testing.T) {
	first, second, stopped := stage(stages.StatusPlanning), stage(stages.StatusStandBy), stage(stages.StatusStopped)
	own := stage(stages.StatusPlanning)
	own.AddReason, own.AddRequestedBy = "Petición", "PM"
	repo := &fakeRepo{project: projects.Project{ID: uuid.New(), Status: projects.StatusPlanning}, stages: []stages.Stage{first, second, stopped, own}}

	result, err := New(repo, fakeTasks{}).Handle(context.Background(), Command{ID: repo.project.ID, Reason: "listo", Actor: pp})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != projects.StatusStandBy || len(result.Stages) != 2 {
		t.Fatalf("result = %+v", result)
	}
	plan := repo.plan
	if plan.Project.Change.To != projects.StatusStandBy || plan.Project.Change.ActorName != "pp" || plan.Project.Change.Reason != "listo" {
		t.Errorf("registro del proyecto = %+v", plan.Project.Change)
	}
	if len(plan.Stages) != 2 || plan.Stages[0].Stage.ID != first.ID || plan.Stages[0].Add != nil {
		t.Fatalf("etapas = %+v", plan.Stages)
	}
	if plan.Stages[1].Stage.ID != own.ID || plan.Stages[1].Add == nil || plan.Stages[1].Add.Reason != "Petición" {
		t.Errorf("el planning propio sale con su add: %+v", plan.Stages[1])
	}
}

func TestHandleRejects(t *testing.T) {
	empty := stage(stages.StatusPlanning)
	noTasks := kernel.Conflict("sin tareas")

	tests := []struct {
		name    string
		project projects.Status
		stages  []stages.Stage
		tasks   fakeTasks
		actor   kernel.Actor
		want    error
	}{
		{"proyecto fuera de planning", projects.StatusStandBy, []stages.Stage{stage(stages.StatusPlanning)}, fakeTasks{}, pp, projects.ErrProjectNotPlanning},
		{"perfil", projects.StatusPlanning, []stages.Stage{stage(stages.StatusPlanning)}, fakeTasks{}, kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}, kernel.ErrTransitionForbidden},
		{"sin etapas", projects.StatusPlanning, nil, fakeTasks{}, pp, stages.ErrNoActiveStages},
		{"solo canceladas", projects.StatusPlanning, []stages.Stage{stage(stages.StatusCancelled)}, fakeTasks{}, pp, stages.ErrNoActiveStages},
		{"regla 4 en una etapa", projects.StatusPlanning, []stages.Stage{stage(stages.StatusPlanning), empty}, fakeTasks{failFor: empty.ID, err: noTasks}, pp, noTasks},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{project: projects.Project{ID: uuid.New(), Status: tt.project}, stages: tt.stages}

			_, err := New(repo, tt.tasks).Handle(context.Background(), Command{ID: repo.project.ID, Reason: "x", Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if repo.plan != nil {
				t.Error("no debió guardar")
			}
		})
	}
}

func TestStageErrorNamesTheStage(t *testing.T) {
	empty := stage(stages.StatusPlanning)
	empty.Name = "Pagos"
	repo := &fakeRepo{project: projects.Project{ID: uuid.New(), Status: projects.StatusPlanning}, stages: []stages.Stage{empty}}

	_, err := New(repo, fakeTasks{failFor: empty.ID, err: kernel.Conflict("sin tareas")}).Handle(context.Background(), Command{ID: repo.project.ID, Actor: pp})

	if err == nil || !strings.Contains(err.Error(), "Pagos") {
		t.Fatalf("err = %v", err)
	}
}
