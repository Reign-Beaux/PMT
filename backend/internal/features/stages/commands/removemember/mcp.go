package removemember

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
	Agent   string `json:"agent" jsonschema:"Nombre del dev"`
}

type toolOutput struct {
	ID      string   `json:"id"`
	Lead    string   `json:"lead"`
	Members []string `json:"members"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "remove_stage_member",
		Description: "Quita un dev del equipo de una etapa de PMT. Solo el TL de la etapa. Se rechaza si el dev tiene tareas abiertas en ella: reasígnalas antes con assign_task.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: stageID, Agent: input.Agent, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Lead: result.Lead, Members: result.Members}, nil
}
