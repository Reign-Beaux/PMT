package domain

import (
	"strings"

	"pmt/internal/kernel"
)

var (
	// ErrBugNotTaken indica que el bug no está tomado y no hay nada que liberar.
	ErrBugNotTaken = kernel.Conflict("el bug no está tomado")
	// ErrReleaseForbidden indica que el perfil no puede liberar un bug.
	ErrReleaseForbidden = kernel.Forbidden("tu perfil no puede liberar un bug")
	// ErrReleaseNotOwner indica que solo el owner del proyecto libera un bug.
	ErrReleaseNotOwner = kernel.Forbidden("solo el owner del proyecto libera un bug tomado")
	// ErrReleaseReasonRequired indica que liberar un bug exige motivo.
	ErrReleaseReasonRequired = kernel.Invalid("liberar un bug exige motivo")
	// ErrBugHoldChanged indica que otra operación tomó o liberó el bug al mismo tiempo.
	ErrBugHoldChanged = kernel.Conflict("el bug cambió de manos, vuelve a consultar")
)

// ErrTakenBy es la respuesta a quien escribe en un bug que tiene tomado otra persona.
func ErrTakenBy(username string) error {
	if username == "" {
		return kernel.Conflict("el bug lo está trabajando otra persona")
	}
	return kernel.Conflict("el bug lo está trabajando " + username)
}

// applyHold toma o libera el bug según el status al que acaba de pasar: lo toma
// quien lo pasa a in_dev, también al retomarlo desde changes_requested, y se
// libera en done o rejected. De in_dev a pending_close sigue tomado.
func (b *Bug) applyHold(actor kernel.Actor) {
	switch b.Status {
	case StatusInDev:
		b.Hold = kernel.HoldOf(actor)
	case StatusDone, StatusRejected, StatusReported, StatusConfirmed:
		b.Hold = kernel.Hold{}
	}
}

// EnsureCanRelease devuelve un error si el perfil no puede liberar un bug: solo
// PP y User, y además debe ser el owner del proyecto.
func EnsureCanRelease(actor kernel.Actor) error {
	if !actor.Profile.IsPlanner() {
		return ErrReleaseForbidden
	}
	return nil
}

// Release deja el bug sin tomar y devuelve el registro para su historial de
// status: mismo status de origen y destino, con el motivo y quién lo tenía. El
// siguiente que escriba no lo toma: solo lo toma quien lo pase a in_dev.
func (b *Bug) Release(actor kernel.Actor, reason string) (StatusChange, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return StatusChange{}, ErrReleaseReasonRequired
	}
	if !b.Taken() {
		return StatusChange{}, ErrBugNotTaken
	}
	note := "liberado por el owner (lo tenía " + b.TakenBy + "): " + reason
	b.Hold = kernel.Hold{}
	return NewStatusChange(b.ID, b.Status, b.Status, actor, note), nil
}
