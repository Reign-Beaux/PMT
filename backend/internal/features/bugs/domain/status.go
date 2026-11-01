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
	// StatusInDev indica que está en corrección.
	StatusInDev Status = "in_dev"
	// StatusInReview indica que está en revisión de código.
	StatusInReview Status = "in_review"
	// StatusChangesRequested indica que tiene correcciones pendientes.
	StatusChangesRequested Status = "changes_requested"
	// StatusInQA indica que está en verificación de QA.
	StatusInQA Status = "in_qa"
	// StatusPendingClose indica que fue aprobado y falta el cierre del TL.
	StatusPendingClose Status = "pending_close"
	// StatusDone indica que está corregido y verificado.
	StatusDone Status = "done"
	// StatusRejected indica que no se corregirá: no se reproduce, está duplicado o no es un bug.
	StatusRejected Status = "rejected"
)

var (
	// ErrBugStatusInvalid indica que el status no es uno de los valores conocidos.
	ErrBugStatusInvalid = kernel.Invalid("status desconocido: usa reported, confirmed, in_dev, in_review, changes_requested, in_qa, pending_close, done o rejected")
	// ErrBugStatusTransition indica que el status actual no permite pasar al pedido.
	ErrBugStatusTransition = kernel.Conflict("transición de status no permitida")
)

// statusTransitions indica a qué estados puede pasar cada estado.
var statusTransitions = map[Status][]Status{
	StatusReported:         {StatusConfirmed, StatusRejected},
	StatusConfirmed:        {StatusInDev, StatusRejected},
	StatusInDev:            {StatusInReview, StatusRejected},
	StatusInReview:         {StatusInQA, StatusChangesRequested, StatusPendingClose, StatusRejected},
	StatusChangesRequested: {StatusInDev, StatusRejected},
	StatusInQA:             {StatusPendingClose, StatusChangesRequested, StatusRejected},
	StatusPendingClose:     {StatusDone, StatusInReview, StatusInQA, StatusRejected},
	StatusDone:             {},
	StatusRejected:         {StatusReported},
}

// ChangeStatus pasa el bug a otro estado si la transición está permitida.
func (b *Bug) ChangeStatus(status Status, actor kernel.Actor, reason string) error {
	if _, known := statusTransitions[status]; !known {
		return ErrBugStatusInvalid
	}
	if !slices.Contains(statusTransitions[b.Status], status) {
		return ErrBugStatusTransition
	}
	if !slices.Contains(statusProfiles[b.Status][status], actor.Profile) {
		return kernel.ErrTransitionForbidden
	}
	if err := ensureReason(b.Status, status, reason); err != nil {
		return err
	}
	b.Status = status
	return nil
}
