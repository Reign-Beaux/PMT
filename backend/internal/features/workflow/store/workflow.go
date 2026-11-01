// Package store implementa la persistencia de la feature workflow: usa el store de
// projects, stages y tasks sobre una sola transacción, cuyos BeginFunc internos son
// savepoints. El bloqueo sigue el orden proyecto, etapa y tareas por ID.
package store

import (
	"cmp"
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	projectdomain "pmt/internal/features/projects/domain"
	projects "pmt/internal/features/projects/store"
	stagedomain "pmt/internal/features/stages/domain"
	stages "pmt/internal/features/stages/store"
	taskdomain "pmt/internal/features/tasks/domain"
	tasks "pmt/internal/features/tasks/store"
	"pmt/internal/features/workflow/domain"
	"pmt/internal/kernel"
)

// VersionOneClosedReason es el motivo del paso automático a working.
const VersionOneClosedReason = "cierre de la versión 1"

// DB es la base de datos de la feature. pgxpool.Pool la implementa.
type DB interface {
	projects.DB
}

// Workflow guarda las operaciones que cruzan features.
type Workflow struct {
	db DB
}

// NewWorkflow crea el repositorio.
func NewWorkflow(db DB) *Workflow {
	return &Workflow{db: db}
}

// Project busca un proyecto.
func (s *Workflow) Project(ctx context.Context, id uuid.UUID) (projectdomain.Project, error) {
	return projects.NewProjects(s.db).Get(ctx, id)
}

// Stage busca la última versión de una etapa sin retirar.
func (s *Workflow) Stage(ctx context.Context, id uuid.UUID) (stagedomain.Stage, error) {
	return stages.NewStages(s.db).Get(ctx, id)
}

// Stages devuelve las etapas sin retirar de un proyecto.
func (s *Workflow) Stages(ctx context.Context, projectID uuid.UUID) ([]stagedomain.Stage, error) {
	return stages.NewStages(s.db).List(ctx, projectID)
}

// Tasks devuelve las tareas sin retirar de una etapa.
func (s *Workflow) Tasks(ctx context.Context, stageID uuid.UUID) ([]taskdomain.Task, error) {
	return tasks.NewTasks(s.db).List(ctx, stageID)
}

// Release mueve el proyecto y sus etapas. Si algo falla, no guarda nada.
func (s *Workflow) Release(ctx context.Context, plan domain.ReleasePlan) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := projects.NewStatusChanges(tx).ChangeStatus(ctx, plan.Project.Project, plan.Project.Change); err != nil {
			return err
		}
		for _, move := range plan.Stages {
			if err := moveStage(ctx, tx, move); err != nil {
				return err
			}
		}
		return nil
	})
}

// Start mueve la etapa, sus tareas y, si hace falta, el proyecto. Si algo falla,
// no guarda nada.
func (s *Workflow) Start(ctx context.Context, plan domain.StartPlan) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if plan.Project != nil {
			if _, err := projects.NewStatusChanges(tx).ChangeStatus(ctx, plan.Project.Project, plan.Project.Change); err != nil {
				return err
			}
		}
		if err := moveStage(ctx, tx, plan.Stage); err != nil {
			return err
		}
		moves := slices.SortedFunc(slices.Values(plan.Tasks), func(a, b domain.TaskMove) int {
			return cmp.Compare(a.Task.ID.String(), b.Task.ID.String())
		})
		repo := tasks.NewStatusChanges(tx)
		for _, move := range moves {
			if _, err := repo.ChangeStatus(ctx, move.Task, move.Change); err != nil {
				return err
			}
		}
		return nil
	})
}

// CloseStage pasa la etapa a done o a cancelled y, si con ella termina la versión 1 y
// el proyecto sigue en in_dev, lo pasa a working con actor como responsable. Bloquea
// primero el proyecto: dos cierres simultáneos se ven entre sí y uno de ellos lo mueve.
func (s *Workflow) CloseStage(ctx context.Context, stage stagedomain.Stage, change stagedomain.StatusChange, actor kernel.Actor) (stagedomain.Stage, error) {
	var updated stagedomain.Stage
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		project, err := projects.NewProjects(tx).Lock(ctx, stage.ProjectID)
		if err != nil {
			return err
		}
		if updated, err = stages.NewStatusChanges(tx).ChangeStatus(ctx, stage, change); err != nil {
			return err
		}
		versionOne, err := stages.NewStages(tx).ListVersionOne(ctx, stage.ProjectID)
		if err != nil {
			return err
		}
		if !stagedomain.EndsVersionOne(updated, versionOne) {
			return nil
		}
		return closeVersionOne(ctx, tx, project, actor)
	})
	if err != nil {
		return stagedomain.Stage{}, err
	}
	return updated, nil
}

// RetireStage registra el retiro de la etapa y, si con él termina la versión 1 y el
// proyecto sigue en in_dev, lo pasa a working con actor como responsable. Bloquea
// primero el proyecto, igual que CloseStage.
func (s *Workflow) RetireStage(ctx context.Context, stage stagedomain.Stage, change stagedomain.Change, actor kernel.Actor) (stagedomain.Change, error) {
	var saved stagedomain.Change
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		project, err := projects.NewProjects(tx).Lock(ctx, stage.ProjectID)
		if err != nil {
			return err
		}
		versionOne, err := stages.NewStages(tx).ListVersionOne(ctx, stage.ProjectID)
		if err != nil {
			return err
		}
		if saved, err = stages.NewChanges(tx).Retire(ctx, change); err != nil {
			return err
		}
		if !stagedomain.RetirementEndsVersionOne(stage.ID, versionOne) {
			return nil
		}
		return closeVersionOne(ctx, tx, project, actor)
	})
	if err != nil {
		return stagedomain.Change{}, err
	}
	return saved, nil
}

// closeVersionOne pasa el proyecto ya bloqueado a working si sigue en in_dev.
func closeVersionOne(ctx context.Context, tx pgx.Tx, project projectdomain.Project, actor kernel.Actor) error {
	from := project.Status
	if !project.CloseVersionOne() {
		return nil
	}
	change := projectdomain.NewStatusChange(project.ID, from, project.Status, actor, VersionOneClosedReason)
	_, err := projects.NewStatusChanges(tx).ChangeStatus(ctx, project, change)
	return err
}

func moveStage(ctx context.Context, tx pgx.Tx, move domain.StageMove) error {
	var err error
	if move.Add != nil {
		_, err = stages.NewChanges(tx).LeavePlanning(ctx, move.Stage, *move.Add, move.Change)
	} else {
		_, err = stages.NewStatusChanges(tx).ChangeStatus(ctx, move.Stage, move.Change)
	}
	return err
}
