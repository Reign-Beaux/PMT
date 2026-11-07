// Package add registra una etapa nueva al final de un proyecto: en su planning
// propio si el proyecto está en working, o como cambio en cualquier otro status.
package add

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Add(ctx context.Context, stage domain.Stage, change domain.Change) (domain.Stage, domain.Change, error)
	Create(ctx context.Context, s domain.Stage) (domain.Stage, error)
}

// ProjectChecker indica si un proyecto está en working. Lo implementa la
// feature projects, conectada en app.
type ProjectChecker interface {
	IsWorking(ctx context.Context, projectID uuid.UUID) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ProjectID   uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Reason      string    `json:"reason"`
	RequestedBy string    `json:"requested_by"`

	Actor kernel.Actor `json:"-"`
}

// Result es la etapa agregada y su versión, 0 mientras no tenga registro.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	Version     int           `json:"version"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	AddReason      string `json:"add_reason,omitempty"`
	AddRequestedBy string `json:"add_requested_by,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo     Repository
	projects ProjectChecker
}

// New crea el handler.
func New(repo Repository, projects ProjectChecker) *Handler {
	return &Handler{repo: repo, projects: projects}
}

// Handle agrega la etapa. Con el proyecto en working pospone el registro hasta
// que la etapa salga de su planning propio; si no, lo escribe de inmediato.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanPlan(cmd.Actor); err != nil {
		return Result{}, err
	}
	working, err := h.projects.IsWorking(ctx, cmd.ProjectID)
	if err != nil {
		return Result{}, err
	}
	if working {
		return h.addOwnPlanning(ctx, cmd)
	}

	stage, err := domain.NewStage(cmd.ProjectID, cmd.Name, cmd.Description)
	if err != nil {
		return Result{}, err
	}
	change, err := domain.NewChange(stage, domain.ChangeAdd, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}

	created, change, err := h.repo.Add(ctx, stage, change)
	if err != nil {
		return Result{}, err
	}
	return toResult(created, change.Version), nil
}

func (h *Handler) addOwnPlanning(ctx context.Context, cmd Command) (Result, error) {
	stage, err := domain.NewOwnPlanningStage(cmd.ProjectID, cmd.Name, cmd.Description, cmd.Reason, cmd.RequestedBy)
	if err != nil {
		return Result{}, err
	}
	created, err := h.repo.Create(ctx, stage)
	if err != nil {
		return Result{}, err
	}
	return toResult(created, 0), nil
}

func toResult(s domain.Stage, version int) Result {
	return Result{
		ID:             s.ID,
		ProjectID:      s.ProjectID,
		Name:           s.Name,
		Description:    s.Description,
		Position:       s.Position,
		Status:         s.Status,
		Version:        version,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		AddReason:      s.AddReason,
		AddRequestedBy: s.AddRequestedBy,
	}
}
