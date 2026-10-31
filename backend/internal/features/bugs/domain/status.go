package domain

import (
	"slices"

	"pmt/internal/kernel"
)

// Status es el estado de un bug.
type Status string

const (
	// StatusReported es el estado inicial: reportado, aún sin confirmar.
	StatusReported Status = "reported"
	// StatusConfirmed indica que se reprodujo y falta corregirlo.
	StatusConfirmed Status = "confirmed"
	// StatusInProgress indica que está en corrección.
	StatusInProgress Status = "in_progress"
	// StatusInReview indica que está en revisión de código.
	StatusInReview Status = "in_review"
	// StatusChangesRequested indica que tiene correcciones pendientes.
	StatusChangesRequested Status = "changes_requested"
	// StatusInQA indica que está en verificación de QA.
	StatusInQA Status = "in_qa"
	// StatusDone indica que está corregido y verificado.
	StatusDone Status = "done"
	// StatusRejected indica que no se corregirá: no se reproduce, está duplicado o no es un bug.
	StatusRejected Status = "rejected"
)

var (
	// ErrBugStatusInvalid indica que el status no es uno de los valores conocidos.
	ErrBugStatusInvalid = kernel.Invalid("status desconocido: usa reported, confirmed, in_progress, in_review, changes_requested, in_qa, done o rejected")
	// ErrBugStatusTransition indica que el status actual no permite pasar al pedido.
	ErrBugStatusTransition = kernel.Conflict("transición de status no permitida")
)

// statusTransitions indica a qué estados puede pasar cada estado.
var statusTransitions = map[Status][]Status{
	StatusReported:         {StatusConfirmed, StatusRejected},
	StatusConfirmed:        {StatusInProgress, StatusRejected},
	StatusInProgress:       {StatusInReview, StatusRejected},
	StatusInReview:         {StatusInQA, StatusChangesRequested, StatusRejected},
	StatusChangesRequested: {StatusInReview, StatusRejected},
	StatusInQA:             {StatusDone, StatusChangesRequested, StatusRejected},
	StatusDone:             {},
	StatusRejected:         {StatusReported},
}

// ChangeStatus pasa el bug a otro estado si la transición está permitida.
func (b *Bug) ChangeStatus(status Status) error {
	if _, known := statusTransitions[status]; !known {
		return ErrBugStatusInvalid
	}
	if !slices.Contains(statusTransitions[b.Status], status) {
		return ErrBugStatusTransition
	}
	b.Status = status
	return nil
}
