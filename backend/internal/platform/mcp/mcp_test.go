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
		{"negocio", kernel.Invalid("mal"), "mal"},
		{"envuelto", fmt.Errorf("capa: %w", kernel.NotFound("nada")), "nada"},
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
