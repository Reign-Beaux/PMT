package startstage

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/features/workflow/domain"
	"pmt/internal/kernel"
)

var tl = kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}

type fakeRepo struct {
	project projects.Project
	stage   stages.Stage
	tasks   []tasks.Task
	plan    *domain.StartPlan
}

func (r *fakeRepo) Project(context.Context, uuid.UUID) (projects.Project, error) {
	return r.project, nil
}

func (r *fakeRepo) Stage(context.Context, uuid.UUID) (stages.Stage, error) {
	return r.stage, nil
}

func (r *fakeRepo) Tasks(context.Context, uuid.UUID) ([]tasks.Task, error) {
	return r.tasks, nil
}

func (r *fakeRepo) Start(_ context.Context, plan domain.StartPlan) error {
	r.plan = &plan
	return nil
}

func newRepo(project projects.Status, stage stages.Status) *fakeRepo {
	return &fakeRepo{
		project: projects.Project{ID: uuid.New(), Status: project},
		stage:   stages.Stage{ID: uuid.New(), Name: "Pagos", Status: stage, Lead: "tl"},
	}
}

func TestHandleStartsStageTasksAndProject(t *testing.T) {
	repo := newRepo(projects.StatusStandBy, stages.StatusStandBy)
	repo.tasks = []tasks.Task{{ID: uuid.New(), Status: tasks.StatusBacklog}, {ID: uuid.New(), Status: tasks.StatusCancelled}, {ID: uuid.New(), Status: tasks.StatusBacklog}}

	result, err := New(repo).Handle(context.Background(), Command{ID: repo.stage.ID, Reason: "arranca", Actor: tl})

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != stages.StatusInDev || result.ProjectStatus != projects.StatusInDev || len(result.Tasks) != 2 {
		t.Fatalf("result = %+v", result)
	}
	plan := repo.plan
	if plan.Project == nil || plan.Project.Project.Status != projects.StatusInDev {
		t.Fatalf("proyecto = %+v", plan.Project)
	}
	change := plan.Project.Change
	if change.From != projects.StatusStandBy || change.ActorName != "tl" || change.Reason != "inicio de la etapa Pagos" {
		t.Errorf("registro del proyecto = %+v", change)
	}
	if len(plan.Tasks) != 2 || plan.Tasks[0].Change.To != tasks.StatusReady || plan.Tasks[0].Change.Reason != "arranca" {
		t.Errorf("tareas = %+v", plan.Tasks)
	}
}

func TestHandleLeavesProjectAlone(t *testing.T) {
	for _, status := range []projects.Status{projects.StatusInDev, projects.StatusWorking} {
		t.Run(string(status), func(t *testing.T) {
			repo := newRepo(status, stages.StatusStandBy)

			result, err := New(repo).Handle(context.Background(), Command{ID: repo.stage.ID, Actor: tl})

			if err != nil {
				t.Fatal(err)
			}
			if repo.plan.Project != nil || result.ProjectStatus != status {
				t.Errorf("proyecto = %+v, result = %+v", repo.plan.Project, result)
			}
		})
	}
}

func TestHandleRejects(t *testing.T) {
	tests := []struct {
		name    string
		project projects.Status
		stage   stages.Status
		actor   kernel.Actor
		want    error
	}{
		{"etapa en planning", projects.StatusStandBy, stages.StatusPlanning, tl, stages.ErrStageNotStandBy},
		{"etapa detenida", projects.StatusStandBy, stages.StatusStopped, tl, stages.ErrStageNotStandBy},
		{"proyecto en planning", projects.StatusPlanning, stages.StatusStandBy, tl, stages.ErrStageProjectNotReady},
		{"proyecto detenido", projects.StatusStopped, stages.StatusStandBy, tl, stages.ErrStageProjectNotReady},
		{"proyecto cancelado", projects.StatusCancelled, stages.StatusStandBy, tl, stages.ErrStageProjectNotReady},
		{"TL ajeno", projects.StatusStandBy, stages.StatusStandBy, kernel.Actor{Name: "tl2", Profile: kernel.ProfileTL}, kernel.ErrNotStageLead},
		{"usuario", projects.StatusStandBy, stages.StatusStandBy, kernel.Owner, kernel.ErrTransitionForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo(tt.project, tt.stage)

			_, err := New(repo).Handle(context.Background(), Command{ID: repo.stage.ID, Actor: tt.actor})

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, quiero %v", err, tt.want)
			}
			if repo.plan != nil {
				t.Error("no debió guardar")
			}
		})
	}
}
