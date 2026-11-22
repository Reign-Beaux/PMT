// Package holdof resuelve quién tiene tomada la etapa o el bug que contiene a un
// elemento. Es la base de la comprobación de bloqueos de las escrituras.
package holdof

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	HoldOf(ctx context.Context, kind string, id uuid.UUID) (holderKind string, hold kernel.Hold, err error)
}

// Result es lo que contiene al elemento, stage o bug (vacío si no pertenece a
// ninguno), y quién lo tiene tomado.
type Result struct {
	HolderKind string
	Hold       kernel.Hold
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle resuelve el elemento. kind es stage, task, bug, finding, test_plan o
// dev_plan; un proyecto no pertenece a ninguna etapa ni bug.
func (h *Handler) Handle(ctx context.Context, kind string, id uuid.UUID) (Result, error) {
	holderKind, hold, err := h.reader.HoldOf(ctx, kind, id)
	if err != nil {
		return Result{}, err
	}
	return Result{HolderKind: holderKind, Hold: hold}, nil
}
