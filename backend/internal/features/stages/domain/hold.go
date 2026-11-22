package domain

import (
	"strings"

	"pmt/internal/kernel"
)

var (
	// ErrStageNotTaken indica que la etapa no está tomada y no hay nada que liberar.
	ErrStageNotTaken = kernel.Conflict("la etapa no está tomada")
	// ErrReleaseForbidden indica que el perfil no puede liberar una etapa.
	ErrReleaseForbidden = kernel.Forbidden("tu perfil no puede liberar una etapa")
	// ErrReleaseNotOwner indica que solo el owner del proyecto libera una etapa.
	ErrReleaseNotOwner = kernel.Forbidden("solo el owner del proyecto libera una etapa tomada")
	// ErrReleaseReasonRequired indica que liberar una etapa exige motivo.
	ErrReleaseReasonRequired = kernel.Invalid("liberar una etapa exige motivo")
	// ErrStageHoldChanged indica que otra operación tomó o liberó la etapa al mismo tiempo.
	ErrStageHoldChanged = kernel.Conflict("la etapa cambió de manos, vuelve a consultar")
)

// ErrTakenBy es la respuesta a quien escribe en una etapa, o en sus tareas, que
// tiene tomada otra persona.
func ErrTakenBy(username string) error {
	if username == "" {
		return kernel.Conflict("la etapa la está trabajando otra persona")
	}
	return kernel.Conflict("la etapa la está trabajando " + username)
}

// applyHold toma o libera la etapa según el status al que acaba de pasar: la
// toma quien la pasa a in_dev (al iniciarla, reanudarla o reabrirla desde done) y
// se libera al cerrarla, cancelarla o volver a planning o stand_by, porque una
// etapa sin iniciar no está tomada. Detenida sigue tomada.
func (s *Stage) applyHold(actor kernel.Actor) {
	switch s.Status {
	case StatusInDev:
		s.Hold = kernel.HoldOf(actor)
	case StatusDone, StatusCancelled, StatusPlanning, StatusStandBy:
		s.Hold = kernel.Hold{}
	}
}

// EnsureCanRelease devuelve un error si el perfil no puede liberar una etapa:
// solo PP y User, y además debe ser el owner del proyecto.
func EnsureCanRelease(actor kernel.Actor) error {
	if !actor.Profile.IsPlanner() {
		return ErrReleaseForbidden
	}
	return nil
}

// Release deja la etapa sin tomar y devuelve el registro para su historial de
// status: mismo status de origen y destino, con el motivo y quién la tenía. El
// siguiente que escriba no la toma: solo la toman start_stage o reactivarla.
func (s *Stage) Release(actor kernel.Actor, reason string) (StatusChange, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return StatusChange{}, ErrReleaseReasonRequired
	}
	if !s.Taken() {
		return StatusChange{}, ErrStageNotTaken
	}
	note := "liberada por el owner (la tenía " + s.TakenBy + "): " + reason
	s.Hold = kernel.Hold{}
	return NewStatusChange(s.ID, s.Status, s.Status, actor, note), nil
}
