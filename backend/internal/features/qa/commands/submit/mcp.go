package submit

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	PlanID string `json:"plan_id" jsonschema:"UUID del plan"`
}

type toolOutput struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "submit_test_plan",
		Description: "Envía un plan de pruebas en draft al TL de la etapa. Solo QAP. happy_path y error_cases no pueden estar vacíos. Un plan devuelto exige antes una versión nueva con revise_test_plan. Usa list_test_plans para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.PlanID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("plan_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Status: string(result.Status), Version: result.Version}, nil
}
