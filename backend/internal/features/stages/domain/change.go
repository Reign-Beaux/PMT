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
	// ErrStageHasChanges indica que la etapa solo se modifica mediante cambios.
	ErrStageHasChanges = kernel.Conflict("la etapa tiene historial de cambios")
)

// ChangeKind es el tipo de un cambio.
type ChangeKind string

const (
	// ChangeAdd indica que la etapa nació como cambio.
	ChangeAdd ChangeKind = "add"
	// ChangeEdit indica una nueva versión de la etapa.
	ChangeEdit ChangeKind = "edit"
	// ChangeRetire indica que la etapa se retiró.
	ChangeRetire ChangeKind = "retire"
)

// Change es un cambio inmutable de una etapa, con su contenido completo. Es una
// entidad del agregado Stage: no tiene métodos que lo modifiquen.
type Change struct {
	ID          uuid.UUID
	StageID     uuid.UUID
	Version     int
	Kind        ChangeKind
	Name        string
	Description string
	Reason      string
	RequestedBy string
	ChangedAt   time.Time
}

// NewChange crea un cambio con el contenido de la etapa. La versión se asigna al guardarlo.
func NewChange(stage Stage, kind ChangeKind, reason, requestedBy string) (Change, error) {
	reason, requestedBy, err := cleanRequest(reason, requestedBy)
	if err != nil {
		return Change{}, err
	}
	return Change{
		ID:          uuid.New(),
		StageID:     stage.ID,
		Kind:        kind,
		Name:        stage.Name,
		Description: stage.Description,
		Reason:      reason,
		RequestedBy: requestedBy,
	}, nil
}

func cleanRequest(reason, requestedBy string) (string, string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", "", ErrChangeReasonRequired
	}
	requestedBy = strings.TrimSpace(requestedBy)
	if requestedBy == "" {
		return "", "", ErrChangeRequesterRequired
	}
	return reason, requestedBy, nil
}

// History es la evolución de una etapa: el contenido original de su fila y sus
// cambios en orden de versión.
type History struct {
	StageID     uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	Changes     []Change
}
