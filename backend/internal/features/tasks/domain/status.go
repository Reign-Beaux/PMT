package domain

import "pmt/internal/kernel"

var (
	// ErrTaskStatusInvalid indica que el status no es uno de los valores conocidos.
	ErrTaskStatusInvalid = kernel.Invalid("status desconocido: usa backlog, ready, in_progress, in_review, changes_requested, in_qa, pending_close, done o cancelled")
	// ErrTaskStatusTransition indica que el status actual no permite pasar al pedido.
	ErrTaskStatusTransition = kernel.Conflict("transición de status no permitida")
)

// statusTransitions indica a qué estados puede pasar cada estado.
var statusTransitions = map[Status][]Status{
	StatusBacklog:          {StatusReady, StatusCancelled},
	StatusReady:            {StatusInProgress, StatusCancelled},
	StatusInProgress:       {StatusInReview, StatusCancelled},
	StatusInReview:         {StatusInQA, StatusChangesRequested, StatusCancelled},
	StatusChangesRequested: {StatusInReview, StatusCancelled},
	StatusInQA:             {StatusPendingClose, StatusChangesRequested, StatusCancelled},
	StatusPendingClose:     {StatusDone, StatusCancelled},
	StatusDone:             {},
	StatusCancelled:        {StatusBacklog},
}

// IsKnownStatus indica si el valor es un status de tarea.
func IsKnownStatus(status Status) bool {
	_, known := statusTransitions[status]
	return known
}

// ChangeStatus pasa la tarea a otro estado si la transición está permitida y
// quien actúa puede hacerla. Tomarla la deja a nombre del dev; reabrirla la libera.
func (t *Task) ChangeStatus(status Status, actor kernel.Actor, reason string, team kernel.Team) error {
	if err := t.EnsureProfile(status, actor); err != nil {
		return err
	}
	if err := t.ensureScope(actor, team); err != nil {
		return err
	}
	if err := ensureReason(t.Status, status, reason); err != nil {
		return err
	}
	switch {
	case t.Status == StatusReady && status == StatusInProgress:
		t.Assignee = actor.Name
	case status == StatusBacklog:
		t.Assignee = ""
	}
	t.Status = status
	return nil
}
