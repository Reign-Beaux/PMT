package listversions

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	PlanID string `json:"plan_id" jsonschema:"UUID del plan"`
}

type toolVersion struct {
	Version            int    `json:"version"`
	Scope              string `json:"scope"`
	TechnicalDecisions string `json:"technical_decisions"`
	ScreenStates       string `json:"screen_states"`
	EdgeCases          string `json:"edge_cases"`
	ErrorHandling      string `json:"error_handling"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Reason             string `json:"reason,omitempty"`
	CreatedBy          string `json:"created_by"`
	CreatedAt          string `json:"created_at"`
}

type toolOutput struct {
	Versions []toolVersion `json:"versions"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_dev_plan_versions",
		Description: "Lista todas las versiones de un plan de desarrollo, en orden. Son inmutables. Disponible para todos los perfiles. Un ID sin versiones responde lista vacía. Usa list_dev_plans para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.PlanID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("plan_id inválido"))
	}

	items, err := h.Handle(ctx, id)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Versions: make([]toolVersion, 0, len(items))}
	for _, item := range items {
		output.Versions = append(output.Versions, toolVersion{
			Version:            item.Version,
			Scope:              item.Scope,
			TechnicalDecisions: item.TechnicalDecisions,
			ScreenStates:       item.ScreenStates,
			EdgeCases:          item.EdgeCases,
			ErrorHandling:      item.ErrorHandling,
			AcceptanceCriteria: item.AcceptanceCriteria,
			Reason:             item.Reason,
			CreatedBy:          item.CreatedBy,
			CreatedAt:          item.CreatedAt.Format(time.RFC3339),
		})
	}
	return nil, output, nil
}
