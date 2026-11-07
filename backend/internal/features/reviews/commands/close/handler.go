// Package close cierra un hallazgo como corregido o descartado.
package close

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Finding, error)
	Close(ctx context.Context, f domain.Finding) (domain.Finding, error)
}

// TaskChecker da la etapa de una tarea, o NotFound si se retiró o se eliminó. Lo
// implementa la feature tasks.
type TaskChecker interface {
	StageOf(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// StageChecker da el equipo de una etapa. Lo implementa la feature stages.
type StageChecker interface {
	Team(ctx context.Context, stageID uuid.UUID) (kernel.Team, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID         uuid.UUID     `json:"-"`
	Resolution domain.Status `json:"resolution"`
	Reason     string        `json:"reason"`

	Actor kernel.Actor `json:"-"`
}

// Result es el hallazgo cerrado.
type Result struct {
	ID           uuid.UUID       `json:"id"`
	ItemKind     domain.ItemKind `json:"item_kind"`
	ItemID       uuid.UUID       `json:"item_id"`
	Description  string          `json:"description"`
	Blocking     bool            `json:"blocking"`
	Status       domain.Status   `json:"status"`
	ActorName    string          `json:"actor_name"`
	ActorProfile kernel.Profile  `json:"actor_profile"`
	ClosedBy     string          `json:"closed_by,omitempty"`
	CloseReason  string          `json:"close_reason,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	ClosedAt     *time.Time      `json:"closed_at,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	tasks  TaskChecker
	stages StageChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker, stages StageChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks, stages: stages}
}

// Handle cierra el hallazgo. Valida resolución, que siga abierto, perfil, ámbito y
// motivo, en ese orden. El ámbito solo se consulta si el perfil ya pasó.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanClose(cmd.Resolution, cmd.Actor); err != nil {
		return Result{}, err
	}
	f, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	if err := f.EnsureClosable(cmd.Resolution, cmd.Actor); err != nil {
		return Result{}, err
	}
	if f.NeedsStageLead(cmd.Resolution) {
		if err := h.ensureStageLead(ctx, f.ItemID, cmd.Actor); err != nil {
			return Result{}, err
		}
	}
	if err := f.Close(cmd.Resolution, cmd.Actor, cmd.Reason); err != nil {
		return Result{}, err
	}

	closed, err := h.repo.Close(ctx, f)
	if err != nil {
		return Result{}, err
	}
	return Result(closed), nil
}

// ensureStageLead exige al TL de la etapa. Si la tarea ya no existe, tampoco se sabe
// quién era su TL, así que basta el perfil, ya validado.
func (h *Handler) ensureStageLead(ctx context.Context, taskID uuid.UUID, actor kernel.Actor) error {
	stageID, err := h.tasks.StageOf(ctx, taskID)
	var business *kernel.Error
	if errors.As(err, &business) && business.Kind == kernel.KindNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	team, err := h.stages.Team(ctx, stageID)
	if err != nil {
		return err
	}
	if !team.IsLead(actor) {
		return kernel.ErrNotStageLead
	}
	return nil
}
