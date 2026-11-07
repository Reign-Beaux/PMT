// Package domain contiene los conceptos y reglas de la feature activity.
package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// ItemKind es el tipo de elemento al que pertenece un comentario.
type ItemKind string

const (
	// ItemProject indica que el comentario es de un proyecto.
	ItemProject ItemKind = "project"
	// ItemStage indica que el comentario es de una etapa.
	ItemStage ItemKind = "stage"
	// ItemTask indica que el comentario es de una tarea.
	ItemTask ItemKind = "task"
	// ItemBug indica que el comentario es de un bug.
	ItemBug ItemKind = "bug"
)

var (
	// ErrItemKindInvalid indica que el tipo de elemento no es uno de los conocidos.
	ErrItemKindInvalid = kernel.Invalid("item_kind desconocido: usa project, stage, task o bug")
	// ErrBodyRequired indica que el comentario está vacío.
	ErrBodyRequired = kernel.Invalid("el texto del comentario es obligatorio")
	// ErrCommentForbidden indica que el perfil no puede comentar.
	ErrCommentForbidden = kernel.Forbidden("PC solo lee: no comenta")
)

// Comment es un comentario inmutable sobre un elemento, al que referencia por ID, sin
// FK. Guarda una copia del actor para conservar el perfil que tenía al comentar.
type Comment struct {
	ID           uuid.UUID
	ItemKind     ItemKind
	ItemID       uuid.UUID
	Body         string
	ActorName    string
	ActorProfile kernel.Profile
	CreatedAt    time.Time
}

// EnsureKind devuelve un error de negocio si el tipo de elemento no es conocido.
func EnsureKind(kind ItemKind) error {
	switch kind {
	case ItemProject, ItemStage, ItemTask, ItemBug:
		return nil
	}
	return ErrItemKindInvalid
}

// NewComment crea un comentario. Comenta cualquier perfil menos PC, que solo lee.
// Solo espacios cuenta como vacío.
func NewComment(kind ItemKind, itemID uuid.UUID, body string, actor kernel.Actor) (Comment, error) {
	if actor.Profile == kernel.ProfilePC {
		return Comment{}, ErrCommentForbidden
	}
	if err := EnsureKind(kind); err != nil {
		return Comment{}, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return Comment{}, ErrBodyRequired
	}
	return Comment{
		ID:           uuid.New(),
		ItemKind:     kind,
		ItemID:       itemID,
		Body:         body,
		ActorName:    actor.Name,
		ActorProfile: actor.Profile,
	}, nil
}
