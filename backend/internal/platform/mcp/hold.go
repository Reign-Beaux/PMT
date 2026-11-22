package mcp

import (
	"time"

	"pmt/internal/kernel"
)

// HoldFields da los campos taken_by y taken_at de una etapa o un bug en la salida
// de una herramienta: vacíos si nadie lo tiene tomado.
func HoldFields(hold kernel.Hold) (takenBy, takenAt string) {
	if !hold.Taken() {
		return "", ""
	}
	return hold.TakenBy, hold.TakenAt.Format(time.RFC3339)
}
