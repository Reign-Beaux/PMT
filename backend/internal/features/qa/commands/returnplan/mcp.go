package returnplan

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	PlanID      string `json:"plan_id" jsonschema:"UUID del plan"`
	ReviewNotes string `json:"review_notes" jsonschema:"Correcciones que debe hacer QAP"`
}

type toolOutput struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
	ReviewNotes string `json:"review_notes"`
	ReviewedBy  string `json:"reviewed_by"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "return_test_plan",
		Description: "Devuelve un plan de pruebas enviado a QAP, con correcciones obligatorias. Solo el TL de la etapa. QAP debe crear una versión nueva con revise_test_plan antes de volver a enviarlo. Usa list_test_plans para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.PlanID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("plan_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, ReviewNotes: input.ReviewNotes, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Status: string(result.Status), Version: result.Version, ReviewNotes: result.ReviewNotes, ReviewedBy: result.ReviewedBy}, nil
}
