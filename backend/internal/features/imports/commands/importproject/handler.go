// Package importproject crea un proyecto completo desde un fichero YAML.
package importproject

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"pmt/internal/features/imports/domain"
	"pmt/internal/features/imports/file"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Save(ctx context.Context, plan domain.Plan) error
}

// Command contiene los datos de entrada.
type Command struct {
	Path string
}

// Result resume lo creado.
type Result struct {
	ProjectID   uuid.UUID
	ProjectName string
	StageCount  int
	TaskCount   int
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle valida el fichero completo y después guarda todo como una unidad.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	entry, err := file.ReadProject(cmd.Path)
	if err != nil {
		return Result{}, err
	}

	plan, err := buildPlan(entry)
	if err != nil {
		return Result{}, err
	}

	if err := h.repo.Save(ctx, plan); err != nil {
		return Result{}, err
	}

	result := Result{
		ProjectID:   plan.Project.ID,
		ProjectName: plan.Project.Name,
		StageCount:  len(plan.Stages),
	}
	for _, stage := range plan.Stages {
		result.TaskCount += len(stage.Tasks)
	}
	return result, nil
}

func buildPlan(entry file.Project) (domain.Plan, error) {
	project, err := projects.NewProject(entry.Name, entry.Description)
	if err != nil {
		return domain.Plan{}, file.Located("proyecto", err.Error())
	}
	if file.Blank(entry.Description) {
		return domain.Plan{}, file.Located("proyecto", file.DescriptionRequired)
	}

	plan := domain.Plan{Project: project}
	for i, stageEntry := range entry.Stages {
		where := fmt.Sprintf("etapa %d", i+1)
		stage, err := stages.NewStage(project.ID, stageEntry.Name, stageEntry.Description)
		if err != nil {
			return domain.Plan{}, file.Located(where, err.Error())
		}
		if file.Blank(stageEntry.Description) {
			return domain.Plan{}, file.Located(where, file.DescriptionRequired)
		}

		tasks, err := stageEntry.BuildTasks(stage.ID, where+", ")
		if err != nil {
			return domain.Plan{}, err
		}
		plan.Stages = append(plan.Stages, domain.StagePlan{Stage: stage, Tasks: tasks})
	}
	return plan, nil
}
