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

// statusTransitions indica a qué estados puede pasar cada estado.
var statusTransitions = map[Status][]Status{
	StatusPlanning:  {StatusStandBy, StatusStopped, StatusCancelled},
	StatusStandBy:   {StatusPlanning, StatusInDev, StatusStopped, StatusCancelled},
	StatusInDev:     {StatusWorking, StatusStopped, StatusCancelled},
	StatusWorking:   {StatusStopped, StatusCancelled},
	StatusStopped:   {StatusPlanning, StatusStandBy, StatusInDev, StatusWorking, StatusCancelled},
	StatusCancelled: {StatusPlanning},
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
