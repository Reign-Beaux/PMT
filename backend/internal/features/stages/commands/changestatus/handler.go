// Package changestatus cambia el status de una etapa.
package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Stage, error)
	ChangeStatus(ctx context.Context, s domain.Stage, change domain.StatusChange) (domain.Stage, error)
	LeavePlanning(ctx context.Context, s domain.Stage, change domain.Change, statusChange domain.StatusChange) (domain.Stage, error)
	// CloseStage guarda el paso a done o a cancelled y, si termina la versión 1, pasa
	// el proyecto a working en la misma transacción. Lo implementa la feature workflow.
	CloseStage(ctx context.Context, s domain.Stage, change domain.StatusChange, actor kernel.Actor) (domain.Stage, error)
}

// TaskChecker comprueba que las tareas de una etapa permitan cerrarla o liberarla
// y responde con el error de negocio. Lo implementa la feature tasks.
type TaskChecker interface {
	EnsureClosable(ctx context.Context, stageID uuid.UUID) error
	EnsureReleasable(ctx context.Context, stageID uuid.UUID) error
}

// ProjectChecker indica qué admite un proyecto según su status. Lo implementa la
// feature projects.
type ProjectChecker interface {
	AcceptsWork(ctx context.Context, projectID uuid.UUID) (bool, error)
	IsFrozen(ctx context.Context, projectID uuid.UUID) (bool, error)
}

// Command contiene los datos de entrada.
type Command struct {
	ID     uuid.UUID     `json:"-"`
	Status domain.Status `json:"status"`
	Reason string        `json:"reason"`
	Actor  kernel.Actor  `json:"-"`
}

// Result es la etapa con su nuevo status.
type Result struct {
	ID          uuid.UUID     `json:"id"`
	ProjectID   uuid.UUID     `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	Status      domain.Status `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	AddReason      string `json:"add_reason,omitempty"`
	AddRequestedBy string `json:"add_requested_by,omitempty"`

	Lead    string   `json:"lead,omitempty"`
	Members []string `json:"members,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	repo     Repository
	tasks    TaskChecker
	projects ProjectChecker
}

// New crea el handler.
func New(repo Repository, tasks TaskChecker, projects ProjectChecker) *Handler {
	return &Handler{repo: repo, tasks: tasks, projects: projects}
}

// Handle cambia el status de la etapa si su proyecto y sus tareas lo permiten.
// Al salir de su planning propio escribe el registro add pendiente; ese camino
// cumple las mismas reglas. Todo paso a done o cancelled pasa por CloseStage.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	s, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return Result{}, err
	}
	from := s.Status
	if err := s.ChangeStatus(cmd.Status, cmd.Actor, cmd.Reason); err != nil {
		return Result{}, err
	}
	if err := h.ensureRules(ctx, from, s); err != nil {
		return Result{}, err
	}
	statusChange := domain.NewStatusChange(s.ID, from, s.Status, cmd.Actor, cmd.Reason)
	if s.InOwnPlanning() {
		change, err := domain.NewChange(s, domain.ChangeAdd, s.AddReason, s.AddRequestedBy)
		if err != nil {
			return Result{}, err
		}
		updated, err := h.repo.LeavePlanning(ctx, s, change, statusChange)
		if err != nil {
			return Result{}, err
		}
		return Result(updated), nil
	}
	if s.Status == domain.StatusDone || s.Status == domain.StatusCancelled {
		updated, err := h.repo.CloseStage(ctx, s, statusChange, cmd.Actor)
		if err != nil {
			return Result{}, err
		}
		return Result(updated), nil
	}

	updated, err := h.repo.ChangeStatus(ctx, s, statusChange)
	if err != nil {
		return Result{}, err
	}
	return Result(updated), nil
}

// ensureRules aplica las reglas 4, 5 y 10 y la decisión 8. La 4 solo vale desde
// planning; la 5, desde cualquier origen. Reabrir no consulta las tareas.
func (h *Handler) ensureRules(ctx context.Context, from domain.Status, s domain.Stage) error {
	if s.Status != domain.StatusStopped && s.Status != domain.StatusCancelled {
		frozen, err := h.projects.IsFrozen(ctx, s.ProjectID)
		if err != nil {
			return err
		}
		if frozen {
			return domain.ErrStageProjectFrozen
		}
	}
	if s.Status == domain.StatusInDev {
		accepts, err := h.projects.AcceptsWork(ctx, s.ProjectID)
		if err != nil {
			return err
		}
		if !accepts {
			return domain.ErrStageProjectNotStarted
		}
	}
	if s.Status == domain.StatusDone {
		return h.tasks.EnsureClosable(ctx, s.ID)
	}
	if from == domain.StatusPlanning && s.Status == domain.StatusStandBy {
		return h.tasks.EnsureReleasable(ctx, s.ID)
	}
	return nil
}
