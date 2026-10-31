package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// ErrProjectStatusChanged indica que el status cambió entre leerlo y guardarlo.
var ErrProjectStatusChanged = kernel.Conflict("el status cambió, vuelve a consultar")

// StatusChange es un cambio de status inmutable de un proyecto. Copia el nombre y el
// perfil del actor porque debe conservar el perfil que tenía al actuar.
type StatusChange struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	From         Status
	To           Status
	ActorName    string
	ActorProfile string
	Reason       string
	ChangedAt    time.Time
}

// NewStatusChange crea el registro de un cambio de status. La fecha se asigna al guardarlo.
func NewStatusChange(id uuid.UUID, from, to Status, actor kernel.Actor, reason string) StatusChange {
	return StatusChange{
		ID:           uuid.New(),
		ProjectID:    id,
		From:         from,
		To:           to,
		ActorName:    actor.Name,
		ActorProfile: string(actor.Profile),
		Reason:       strings.TrimSpace(reason),
	}
}
