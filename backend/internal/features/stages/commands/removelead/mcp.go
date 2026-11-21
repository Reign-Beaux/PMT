package removelead

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
	ID      string   `json:"id"`
	Lead    string   `json:"lead"`
	Members []string `json:"members"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "remove_stage_lead",
		Description: "Quita el TL responsable de una etapa de PMT y la deja sin TL. Solo PP y el usuario. Solo con la etapa sin iniciar (planning o stand_by); el equipo no cambia. Para cambiar a otro TL usa assign_stage_lead. Usa list_stages para obtener el ID de la etapa.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: stageID, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Lead: result.Lead, Members: result.Members}, nil
}
