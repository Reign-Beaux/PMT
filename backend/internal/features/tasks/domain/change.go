package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrChangeReasonRequired indica que el motivo del cambio está vacío.
	ErrChangeReasonRequired = kernel.Invalid("el motivo del cambio es obligatorio")
	// ErrChangeRequesterRequired indica que no se indicó quién pidió el cambio.
	ErrChangeRequesterRequired = kernel.Invalid("quién pidió el cambio es obligatorio")
	// ErrTaskHasChanges indica que la tarea solo se modifica mediante cambios.
	ErrTaskHasChanges = kernel.Conflict("la tarea tiene historial de cambios")
)

// ChangeKind es el tipo de un cambio.
type ChangeKind string

const (
	// ChangeAdd indica que la tarea nació como cambio.
	ChangeAdd ChangeKind = "add"
	// ChangeEdit indica una nueva versión de la tarea.
	ChangeEdit ChangeKind = "edit"
	// ChangeRetire indica que la tarea se retiró.
	ChangeRetire ChangeKind = "retire"
)

// Change es un cambio inmutable de una tarea, con su contenido completo. Es una
// entidad del agregado Task: no tiene métodos que lo modifiquen.
type Change struct {
	ID          uuid.UUID
	TaskID      uuid.UUID
	Version     int
	Kind        ChangeKind
	Title       string
	Description string
	Reason      string
	RequestedBy string
	ChangedAt   time.Time
}

// NewChange crea un cambio con el contenido de la tarea. La versión se asigna al guardarlo.
func NewChange(task Task, kind ChangeKind, reason, requestedBy string) (Change, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Change{}, ErrChangeReasonRequired
	}
	requestedBy = strings.TrimSpace(requestedBy)
	if requestedBy == "" {
		return Change{}, ErrChangeRequesterRequired
	}
	return Change{
		ID:          uuid.New(),
		TaskID:      task.ID,
		Kind:        kind,
		Title:       task.Title,
		Description: task.Description,
		Reason:      reason,
		RequestedBy: requestedBy,
	}, nil
}

// History es la evolución de una tarea: el contenido original de su fila y sus
// cambios en orden de versión.
type History struct {
	TaskID      uuid.UUID
	Title       string
	Description string
	CreatedAt   time.Time
	Changes     []Change
}
