// Package domain contiene los conceptos y reglas de la feature reviews.
package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// ItemKind es el tipo de elemento al que pertenece un hallazgo.
type ItemKind string

const (
	// ItemTask indica que el hallazgo es de una tarea.
	ItemTask ItemKind = "task"
	// ItemBug indica que el hallazgo es de un bug.
	ItemBug ItemKind = "bug"
)

// Status es el estado de un hallazgo.
type Status string

const (
	// StatusOpen indica que el hallazgo sigue pendiente.
	StatusOpen Status = "open"
	// StatusFixed indica que se corrigió y se verificó.
	StatusFixed Status = "fixed"
	// StatusDismissed indica que el TL lo descartó.
	StatusDismissed Status = "dismissed"
)

var (
	// ErrFindingNotFound indica que el hallazgo no existe.
	ErrFindingNotFound = kernel.NotFound("hallazgo no encontrado")
	// ErrItemKindInvalid indica que el tipo de elemento no es uno de los conocidos.
	ErrItemKindInvalid = kernel.Invalid("item_kind desconocido: usa task o bug")
	// ErrDescriptionRequired indica que la descripción está vacía.
	ErrDescriptionRequired = kernel.Invalid("la descripción del hallazgo es obligatoria")
	// ErrBlockingRequired indica que no se dijo si el hallazgo es bloqueante.
	ErrBlockingRequired = kernel.Invalid("blocking es obligatorio")
	// ErrResolutionInvalid indica que la resolución no es una de las conocidas.
	ErrResolutionInvalid = kernel.Invalid("resolution desconocida: usa fixed o dismissed")
	// ErrFindingClosed indica que el hallazgo ya se cerró.
	ErrFindingClosed = kernel.Conflict("el hallazgo ya está cerrado")
	// ErrCreateForbidden indica que el perfil no registra hallazgos.
	ErrCreateForbidden = kernel.Forbidden("solo CR o QA registran hallazgos")
	// ErrFixForbidden indica que el perfil no marca hallazgos como corregidos.
	ErrFixForbidden = kernel.Forbidden("solo CR o QA marcan un hallazgo como corregido")
	// ErrDismissForbidden indica que el perfil no descarta hallazgos.
	ErrDismissForbidden = kernel.Forbidden("solo el TL descarta un hallazgo")
)

// Finding es la raíz del agregado de un hallazgo. Referencia a su tarea o bug por
// ID, sin FK, y no toca su status.
type Finding struct {
	ID           uuid.UUID
	ItemKind     ItemKind
	ItemID       uuid.UUID
	Description  string
	Blocking     bool
	Status       Status
	ActorName    string
	ActorProfile kernel.Profile
	ClosedBy     string
	CloseReason  string
	CreatedAt    time.Time
	ClosedAt     *time.Time
}

// EnsureKind devuelve un error de negocio si el tipo de elemento no es conocido.
func EnsureKind(kind ItemKind) error {
	if kind != ItemTask && kind != ItemBug {
		return ErrItemKindInvalid
	}
	return nil
}

// EnsureCanCreate comprueba que el perfil registre hallazgos.
func EnsureCanCreate(actor kernel.Actor) error {
	if !slices.Contains(reviewers, actor.Profile) {
		return ErrCreateForbidden
	}
	return nil
}

// NewFinding crea un hallazgo abierto.
func NewFinding(kind ItemKind, itemID uuid.UUID, description string, blocking bool, actor kernel.Actor) (Finding, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return Finding{}, ErrDescriptionRequired
	}
	return Finding{
		ID:           uuid.New(),
		ItemKind:     kind,
		ItemID:       itemID,
		Description:  description,
		Blocking:     blocking,
		Status:       StatusOpen,
		ActorName:    actor.Name,
		ActorProfile: actor.Profile,
	}, nil
}

// reviewers son los perfiles que registran hallazgos y los dan por corregidos.
var reviewers = []kernel.Profile{kernel.ProfileCR, kernel.ProfileQA}

// EnsureClosable comprueba la resolución, que siga abierto y el perfil, antes de
// consultar el ámbito en otra feature.
func (f Finding) EnsureClosable(resolution Status, actor kernel.Actor) error {
	if resolution != StatusFixed && resolution != StatusDismissed {
		return ErrResolutionInvalid
	}
	if f.Status != StatusOpen {
		return ErrFindingClosed
	}
	if resolution == StatusFixed && !slices.Contains(reviewers, actor.Profile) {
		return ErrFixForbidden
	}
	if resolution == StatusDismissed && actor.Profile != kernel.ProfileTL {
		return ErrDismissForbidden
	}
	return nil
}

// NeedsStageLead indica si cerrarlo exige ser el TL de la etapa de la tarea. Un bug
// no tiene etapa, así que basta el perfil.
func (f Finding) NeedsStageLead(resolution Status) bool {
	return resolution == StatusDismissed && f.ItemKind == ItemTask
}

// Close cierra el hallazgo con su motivo, obligatorio. Solo espacios cuenta como vacío.
func (f *Finding) Close(resolution Status, actor kernel.Actor, reason string) error {
	if err := f.EnsureClosable(resolution, actor); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return kernel.ErrReasonRequired
	}
	f.Status = resolution
	f.ClosedBy = actor.Name
	f.CloseReason = reason
	return nil
}
