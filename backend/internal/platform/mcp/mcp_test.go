package mcp

import (
	"errors"
	"fmt"
	"testing"

	"pmt/internal/kernel"
)

func TestToolError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{"invalid", kernel.Invalid("mal"), "invalid: mal"},
		{"not_found", kernel.NotFound("nada"), "not_found: nada"},
		{"conflict", kernel.Conflict("choca"), "conflict: choca"},
		{"forbidden", kernel.Forbidden("no puedes"), "forbidden: no puedes"},
		{"sin clasificar", &kernel.Error{Message: "mal"}, "invalid: mal"},
		{"envuelto", fmt.Errorf("capa: %w", kernel.NotFound("nada")), "not_found: nada"},
		{"con contexto", kernel.Prefix("tarea 1: ", kernel.Conflict("choca")), "conflict: tarea 1: choca"},
		{"interno", errors.New("secreto"), "error interno"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToolError(tt.err).Error(); got != tt.message {
				t.Errorf("mensaje = %q", got)
			}
		})
	}
}
