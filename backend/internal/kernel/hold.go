package kernel

import (
	"time"

	"github.com/google/uuid"
)

// Hold dice qué persona tiene tomada una etapa o un bug: el usuario cuyos
// agentes, con cualquier perfil, trabajan en él. Los de otra persona solo leen.
// Vacío, nadie lo tiene. TakenAt lo asigna la base al guardarlo.
type Hold struct {
	HolderID uuid.UUID `json:"-"`
	TakenBy  string    `json:"taken_by,omitempty"`
	TakenAt  time.Time `json:"taken_at,omitzero"`
}

// HoldOf es el bloqueo que toma el actor: el de su usuario, que en un agente es
// el dueño de la instancia. Un actor sin usuario no toma nada.
func HoldOf(actor Actor) Hold {
	if actor.UserID == uuid.Nil {
		return Hold{}
	}
	return Hold{HolderID: actor.UserID, TakenBy: actor.UserName}
}

// Taken indica si alguien lo tiene tomado.
func (h Hold) Taken() bool {
	return h.HolderID != uuid.Nil
}

// HeldByOther indica si lo tiene tomado una persona distinta de la del actor.
func (h Hold) HeldByOther(actor Actor) bool {
	return h.Taken() && h.HolderID != actor.UserID
}
