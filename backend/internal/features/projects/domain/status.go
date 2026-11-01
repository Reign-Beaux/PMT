package domain

import (
	"slices"

	"pmt/internal/kernel"
)

var (
	// ErrProjectStatusInvalid indica que el status no es uno de los valores conocidos.
	ErrProjectStatusInvalid = kernel.Invalid("status desconocido: usa planning, stand_by, in_dev, working, stopped o cancelled")
	// ErrProjectStatusTransition indica que el status actual no permite pasar al pedido.
	ErrProjectStatusTransition = kernel.Conflict("transición de status no permitida")
)

// Status es el estado de un proyecto.
type Status string

const (
	// StatusPlanning es el estado inicial de todo proyecto.
	StatusPlanning Status = "planning"
	// StatusStandBy indica que el proyecto está en espera.
	StatusStandBy Status = "stand_by"
	// StatusInDev indica que se desarrolla la primera versión.
	StatusInDev Status = "in_dev"
	// StatusWorking indica que la primera versión ya terminó.
	StatusWorking Status = "working"
	// StatusStopped indica que el proyecto está detenido.
	StatusStopped Status = "stopped"
	// StatusCancelled indica que el proyecto está cancelado.
	StatusCancelled Status = "cancelled"
)

// statusTransitions indica a qué estados puede pasar cada estado. in_dev → working
// no está: solo ocurre al cerrar la versión 1, con CloseVersionOne.
var statusTransitions = map[Status][]Status{
	StatusPlanning:  {StatusStandBy, StatusStopped, StatusCancelled},
	StatusStandBy:   {StatusPlanning, StatusInDev, StatusStopped, StatusCancelled},
	StatusInDev:     {StatusStopped, StatusCancelled},
	StatusWorking:   {StatusStopped, StatusCancelled},
	StatusStopped:   {StatusPlanning, StatusStandBy, StatusInDev, StatusWorking, StatusCancelled},
	StatusCancelled: {StatusPlanning},
}

// StartWork aplica la decisión 2: tomar la primera etapa saca al proyecto de
// stand_by. Es automático, sin tabla de perfiles, e indica si el status cambió.
func (p *Project) StartWork() bool {
	if p.Status != StatusStandBy {
		return false
	}
	p.Status = StatusInDev
	return true
}

// CloseVersionOne aplica la decisión 3: terminar la versión 1 lleva al proyecto de
// in_dev a working. Es automático, sin tabla de perfiles, e indica si el status cambió.
func (p *Project) CloseVersionOne() bool {
	if p.Status != StatusInDev {
		return false
	}
	p.Status = StatusWorking
	return true
}

// ChangeStatus pasa el proyecto a otro estado si la transición está permitida.
func (p *Project) ChangeStatus(status Status, actor kernel.Actor, reason string) error {
	if _, known := statusTransitions[status]; !known {
		return ErrProjectStatusInvalid
	}
	if !slices.Contains(statusTransitions[p.Status], status) {
		return ErrProjectStatusTransition
	}
	if !slices.Contains(statusProfiles[p.Status][status], actor.Profile) {
		return kernel.ErrTransitionForbidden
	}
	if err := ensureReason(p.Status, status, reason); err != nil {
		return err
	}
	p.Status = status
	return nil
}
