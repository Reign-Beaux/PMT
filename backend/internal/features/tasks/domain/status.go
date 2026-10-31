package domain

import (
	"slices"

	"pmt/internal/kernel"
)

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

// ChangeStatus pasa la tarea a otro estado si la transición está permitida.
func (t *Task) ChangeStatus(status Status) error {
	if _, known := statusTransitions[status]; !known {
		return ErrTaskStatusInvalid
	}
	if !slices.Contains(statusTransitions[t.Status], status) {
		return ErrTaskStatusTransition
	}
	t.Status = status
	return nil
}
