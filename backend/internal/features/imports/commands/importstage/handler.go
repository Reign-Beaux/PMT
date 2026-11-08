// Package importstage crea una o varias etapas con sus tareas desde un fichero
// YAML con raíz stages, en un proyecto en planning o en working.
package importstage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"pmt/internal/features/imports/domain"
	"pmt/internal/features/imports/file"
	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	ProjectStatus(ctx context.Context, projectID uuid.UUID) (projects.Status, error)
	SaveStages(ctx context.Context, projectID uuid.UUID, plans []domain.StagePlan) error
}

// Command contiene los datos de entrada. Reason y RequestedBy solo se usan en
// un proyecto en working.
type Command struct {
	ProjectID   uuid.UUID
	Path        string
	Reason      string
	RequestedBy string

	Actor kernel.Actor
}

// StageResult resume una etapa creada.
type StageResult struct {
	StageID   uuid.UUID
	StageName string
	TaskCount int
}

// Result resume lo creado, en el orden del fichero.
type Result struct {
	ProjectID uuid.UUID
	Stages    []StageResult
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
	if err := domain.EnsureCanImport(cmd.Actor); err != nil {
		return Result{}, err
	}
	entries, err := file.ReadStages(cmd.Path)
	if err != nil {
		return Result{}, err
	}
	status, err := h.repo.ProjectStatus(ctx, cmd.ProjectID)
	if err != nil {
		return Result{}, err
	}

	var plans []domain.StagePlan
	switch status {
	case projects.StatusPlanning:
		plans, err = buildPlans(cmd.ProjectID, entries, false, "", "")
	case projects.StatusWorking:
		if file.Blank(cmd.Reason) {
			return Result{}, stages.ErrChangeReasonRequired
		}
		if file.Blank(cmd.RequestedBy) {
			return Result{}, stages.ErrChangeRequesterRequired
		}
		plans, err = buildPlans(cmd.ProjectID, entries, true, cmd.Reason, cmd.RequestedBy)
	default:
		return Result{}, domain.ErrProjectNotImportable
	}
	if err != nil {
		return Result{}, err
	}

	if err := h.repo.SaveStages(ctx, cmd.ProjectID, plans); err != nil {
		return Result{}, err
	}

	result := Result{ProjectID: cmd.ProjectID}
	for _, plan := range plans {
		result.Stages = append(result.Stages, StageResult{
			StageID:   plan.Stage.ID,
			StageName: plan.Stage.Name,
			TaskCount: len(plan.Tasks),
		})
	}
	return result, nil
}

func buildPlans(projectID uuid.UUID, entries []file.Stage, ownPlanning bool, reason, requestedBy string) ([]domain.StagePlan, error) {
	plans := make([]domain.StagePlan, 0, len(entries))
	for i, entry := range entries {
		where := fmt.Sprintf("etapa %d", i+1)

		var stage stages.Stage
		var err error
		if ownPlanning {
			stage, err = stages.NewOwnPlanningStage(projectID, entry.Name, entry.Description, reason, requestedBy)
		} else {
			stage, err = stages.NewStage(projectID, entry.Name, entry.Description)
		}
		if err != nil {
			return nil, file.Located(where, err.Error())
		}
		if file.Blank(entry.Description) {
			return nil, file.Located(where, file.DescriptionRequired)
		}

		tasks, err := entry.BuildTasks(stage.ID, where+", ")
		if err != nil {
			return nil, err
		}
		plans = append(plans, domain.StagePlan{Stage: stage, Tasks: tasks})
	}
	return plans, nil
}
