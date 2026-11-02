// Package mcp proporciona la infraestructura MCP compartida.
package mcp

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

// NewServer crea el servidor MCP de PMT.
func NewServer() *sdk.Server {
	return sdk.NewServer(&sdk.Implementation{Name: "pmt", Version: "0.1.0"}, nil)
}

// Handler expone el servidor por Streamable HTTP. Es sin estado porque cada
// herramienta se resuelve en una sola petición.
func Handler(server *sdk.Server) http.Handler {
	return sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
}

// ToolError prepara un error para el agente. Los de negocio anteponen su tipo
// al mensaje, para que el agente los distinga sin interpretar el texto. Los
// demás se registran y no exponen su detalle.
func ToolError(err error) error {
	var businessErr *kernel.Error
	if errors.As(err, &businessErr) {
		return fmt.Errorf("%s: %w", kindCode(businessErr.Kind), businessErr)
	}
	log.Printf("error interno: %v", err)
	return errors.New("error interno")
}

// kindCode nombra el tipo de un error de negocio. Como en REST, un tipo sin
// clasificar cuenta como entrada inválida.
func kindCode(kind kernel.ErrorKind) string {
	switch kind {
	case kernel.KindNotFound:
		return "not_found"
	case kernel.KindConflict:
		return "conflict"
	case kernel.KindForbidden:
		return "forbidden"
	default:
		return "invalid"
	}
}
