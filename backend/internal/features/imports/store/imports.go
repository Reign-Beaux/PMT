// Package store implementa la persistencia de la feature imports.
package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/imports/domain"
	projectdomain "pmt/internal/features/projects/domain"
	projects "pmt/internal/features/projects/store"
	stages "pmt/internal/features/stages/store"
	tasks "pmt/internal/features/tasks/store"
)

// Beginner abre transacciones. pgxpool.Pool la implementa.
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Imports guarda un plan con los repositorios de cada feature, sobre una sola transacción.
type Imports struct {
	db Beginner
}

// NewImports crea el repositorio.
func NewImports(db Beginner) *Imports {
	return &Imports{db: db}
}

// Save guarda el proyecto, sus etapas y sus tareas en orden. Si algo falla, no guarda nada.
func (s *Imports) Save(ctx context.Context, plan domain.Plan) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		projectRepo := projects.NewProjects(tx)
		stageRepo := stages.NewStages(tx)
		taskRepo := tasks.NewTasks(tx)

		if _, err := projectRepo.Create(ctx, plan.Project); err != nil {
			return err
		}
		for _, stage := range plan.Stages {
			if _, err := stageRepo.Create(ctx, stage.Stage); err != nil {
				return err
			}
			for _, task := range stage.Tasks {
				if _, err := taskRepo.Create(ctx, task); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// SaveStage guarda una etapa y sus tareas en un proyecto en working. Si algo falla, no guarda nada.
func (s *Imports) SaveStage(ctx context.Context, projectID uuid.UUID, plan domain.StagePlan) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		project, err := projects.NewProjects(tx).Get(ctx, projectID)
		if err != nil {
			return err
		}
		if !project.IsWorking() {
			return projectdomain.ErrProjectNotWorking
		}

		if _, err := stages.NewStages(tx).Create(ctx, plan.Stage); err != nil {
			return err
		}
		taskRepo := tasks.NewTasks(tx)
		for _, task := range plan.Tasks {
			if _, err := taskRepo.Create(ctx, task); err != nil {
				return err
			}
		}
		return nil
	})
}
