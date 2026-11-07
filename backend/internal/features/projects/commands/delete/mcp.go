package delete

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto"`
}

type toolOutput struct {
	DeletedID string `json:"deleted_id"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "delete_project",
		Description: "Elimina un proyecto de PMT y devuelve su ID. Solo funciona si el proyecto está en planning y no tiene bugs ni etapas, activas o retiradas. Usa list_projects para obtener el ID del proyecto.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	if err := h.Handle(ctx, id, platformmcp.ActorFrom(req)); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{DeletedID: id.String()}, nil
}
