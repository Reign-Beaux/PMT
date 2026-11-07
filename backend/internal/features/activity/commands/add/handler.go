// Package add registra un comentario sobre un proyecto, una etapa, una tarea o un bug.
package add

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/activity/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, c domain.Comment) (domain.Comment, error)
}

// ProjectChecker confirma que un proyecto exista. Lo implementa la feature projects.
type ProjectChecker interface {
	Check(ctx context.Context, projectID uuid.UUID) error
}

// StageChecker confirma que una etapa exista, sin retirar. Lo implementa la feature stages.
type StageChecker interface {
	Check(ctx context.Context, stageID uuid.UUID) error
}

// TaskChecker confirma que una tarea exista, sin retirar. Lo implementa la feature tasks.
type TaskChecker interface {
	StageOf(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// BugChecker confirma que un bug exista. Lo implementa la feature bugs.
type BugChecker interface {
	Check(ctx context.Context, bugID uuid.UUID) error
}

// Command contiene los datos de entrada.
type Command struct {
	ItemKind domain.ItemKind `json:"item_kind"`
	ItemID   uuid.UUID       `json:"item_id"`
	Body     string          `json:"body"`

	Actor kernel.Actor `json:"-"`
}

// Result es el comentario creado.
type Result struct {
	ID           uuid.UUID       `json:"id"`
	ItemKind     domain.ItemKind `json:"item_kind"`
	ItemID       uuid.UUID       `json:"item_id"`
	Body         string          `json:"body"`
	ActorName    string          `json:"actor_name"`
	ActorProfile kernel.Profile  `json:"actor_profile"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo     Repository
	projects ProjectChecker
	stages   StageChecker
	tasks    TaskChecker
	bugs     BugChecker
}

// New crea el handler.
func New(repo Repository, projects ProjectChecker, stages StageChecker, tasks TaskChecker, bugs BugChecker) *Handler {
	return &Handler{repo: repo, projects: projects, stages: stages, tasks: tasks, bugs: bugs}
}

// Handle registra el comentario. Lo puede hacer cualquier perfil menos PC y no toca el elemento.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	c, err := domain.NewComment(cmd.ItemKind, cmd.ItemID, cmd.Body, cmd.Actor)
	if err != nil {
		return Result{}, err
	}
	if err := h.ensureItemExists(ctx, cmd.ItemKind, cmd.ItemID); err != nil {
		return Result{}, err
	}

	created, err := h.repo.Create(ctx, c)
	if err != nil {
		return Result{}, err
	}
	return Result(created), nil
}

func (h *Handler) ensureItemExists(ctx context.Context, kind domain.ItemKind, id uuid.UUID) error {
	switch kind {
	case domain.ItemProject:
		return h.projects.Check(ctx, id)
	case domain.ItemStage:
		return h.stages.Check(ctx, id)
	case domain.ItemBug:
		return h.bugs.Check(ctx, id)
	}
	_, err := h.tasks.StageOf(ctx, id)
	return err
}
