package get

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

// ErrNoProfile indica que la sesión no declaró perfil.
var ErrNoProfile = kernel.Invalid("esta sesión no tiene perfil: declara uno al conectar el MCP con ?profile=<perfil> en la URL (pp, pc, tl, dp, devbackend, devfrontend, cr, qap o qa)")

type toolInput struct{}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "get_profile",
		Description: "Devuelve, en Markdown, el perfil completo de esta sesión: tus reglas como agente de PMT más las comunes, el mismo texto de GET /api/profiles/{perfil}. Todos los perfiles. Llámala al empezar la sesión, antes de cualquier otra cosa, y sigue lo que dice. Sin perfil declarado (?profile= en la URL del MCP) responde error.",
	}, h.callTool)
}

func (h *Handler) callTool(_ context.Context, req *sdk.CallToolRequest, _ toolInput) (*sdk.CallToolResult, any, error) {
	text, err := h.HandleActor(platformmcp.ActorFrom(req))
	if err != nil {
		return nil, nil, platformmcp.ToolError(err)
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}}, nil, nil
}
