package create

import (
	"context"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	Name        string `json:"name" jsonschema:"Nombre del proyecto"`
	Description string `json:"description" jsonschema:"Descripción del proyecto"`
}

type toolOutput struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	ProductVersion string `json:"product_version"`
	CreatedAt      string `json:"created_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "create_project",
		Description: "Crea un proyecto en PMT. Si el usuario no indicó el nombre o la descripción, pídeselos antes de llamar a esta herramienta.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	result, err := h.Handle(ctx, Command(input))
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:             result.ID.String(),
		Name:           result.Name,
		Description:    result.Description,
		Status:         string(result.Status),
		ProductVersion: result.ProductVersion,
		CreatedAt:      result.CreatedAt.Format(time.RFC3339),
	}, nil
}
