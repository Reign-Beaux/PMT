// Package create registra un hallazgo sobre una tarea o un bug.
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, f domain.Finding) (domain.Finding, error)
}

// TaskChecker confirma que una tarea exista. Lo implementa la feature tasks.
type TaskChecker interface {
	StageOf(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// BugChecker confirma que un bug exista. Lo implementa la feature bugs.
type BugChecker interface {
	Check(ctx context.Context, bugID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ItemKind    domain.ItemKind `json:"item_kind"`
	ItemID      uuid.UUID       `json:"item_id"`
	Description string          `json:"description"`
	Blocking    *bool           `json:"blocking"`

	Actor kernel.Actor `json:"-"`
}

// Result es el hallazgo creado.
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
	repo  Repository
	tasks TaskChecker
	bugs  BugChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker, bugs BugChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks, bugs: bugs}
}

// Handle registra el hallazgo. Solo lo hacen CR y QA, y no toca el status del elemento.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	if err := domain.EnsureCanCreate(cmd.Actor); err != nil {
		return Result{}, err
	}
	if err := domain.EnsureKind(cmd.ItemKind); err != nil {
		return Result{}, err
	}
	if cmd.Blocking == nil {
		return Result{}, domain.ErrBlockingRequired
	}
	f, err := domain.NewFinding(cmd.ItemKind, cmd.ItemID, cmd.Description, *cmd.Blocking, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	if err := h.ensureItemExists(ctx, cmd.ItemKind, cmd.ItemID); err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, f)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}

func (h *Handler) ensureItemExists(ctx context.Context, kind domain.ItemKind, id uuid.UUID) error {
	if kind == domain.ItemBug {
		return h.bugs.Check(ctx, id)
	}
	_, err := h.tasks.StageOf(ctx, id)
	return err
}
