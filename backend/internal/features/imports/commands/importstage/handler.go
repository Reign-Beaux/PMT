// Package importstage crea una etapa completa desde un fichero YAML, en su
// planning propio, dentro de un proyecto en working.
package importstage

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/imports/domain"
	"pmt/internal/features/imports/file"
	stages "pmt/internal/features/stages/domain"
)

const location = "etapa"

// Repository es la persistencia que necesita la operación.
type Repository interface {
	SaveStage(ctx context.Context, projectID uuid.UUID, plan domain.StagePlan) error
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID   uuid.UUID
	Path        string
	Reason      string
	RequestedBy string
}

// Result resume lo creado.
type Result struct {
	ProjectID uuid.UUID
	StageID   uuid.UUID
	StageName string
	TaskCount int
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
	entry, err := file.ReadStage(cmd.Path)
	if err != nil {
		return Result{}, err
	}

	stage, err := stages.NewOwnPlanningStage(cmd.ProjectID, entry.Name, entry.Description, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, file.Located(location, err.Error())
	}
	if file.Blank(entry.Description) {
		return Result{}, file.Located(location, file.DescriptionRequired)
	}
	tasks, err := entry.BuildTasks(stage.ID, "")
	if err != nil {
		return Result{}, err
	}

	if err := h.repo.SaveStage(ctx, cmd.ProjectID, domain.StagePlan{Stage: stage, Tasks: tasks}); err != nil {
		return Result{}, err
	}
	return Result{
		ProjectID: cmd.ProjectID,
		StageID:   stage.ID,
		StageName: stage.Name,
		TaskCount: len(tasks),
	}, nil
}
