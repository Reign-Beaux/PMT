package delete

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
}

type toolOutput struct {
	DeletedID string `json:"deleted_id"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "delete_stage",
		Description: "Elimina una etapa de PMT y devuelve su ID. Solo PP y el usuario. Solo funciona si la etapa no tiene tareas, activas o retiradas, y además está en su planning propio con el proyecto en working, o su proyecto está en planning y la etapa no tiene historial; si no, usa retire_stage. Usa list_stages para obtener el ID de la etapa.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	if err := h.Handle(ctx, id, platformmcp.ActorFrom(req)); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{DeletedID: id.String()}, nil
}
