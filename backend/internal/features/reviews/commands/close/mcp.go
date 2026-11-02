package close

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	FindingID  string `json:"finding_id" jsonschema:"UUID del hallazgo"`
	Resolution string `json:"resolution" jsonschema:"Cierre: fixed o dismissed"`
	Reason     string `json:"reason" jsonschema:"Motivo del cierre"`
}

type toolOutput struct {
	ID          string `json:"id"`
	ItemKind    string `json:"item_kind"`
	ItemID      string `json:"item_id"`
	Status      string `json:"status"`
	ClosedBy    string `json:"closed_by"`
	CloseReason string `json:"close_reason"`
	ClosedAt    string `json:"closed_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "close_finding",
		Description: "Cierra un hallazgo abierto, con motivo obligatorio. fixed: CR o QA, cuando verificaron la corrección. dismissed: el TL de la etapa de la tarea, o un TL si es de un bug o si la tarea se retiró o se eliminó. Un hallazgo cerrado no se vuelve a cerrar ni se reabre. No cambia el status del elemento. Usa list_findings para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.FindingID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("finding_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Resolution: domain.Status(input.Resolution), Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{
		ID:          result.ID.String(),
		ItemKind:    string(result.ItemKind),
		ItemID:      result.ItemID.String(),
		Status:      string(result.Status),
		ClosedBy:    result.ClosedBy,
		CloseReason: result.CloseReason,
	}
	if result.ClosedAt != nil {
		output.ClosedAt = result.ClosedAt.Format(time.RFC3339)
	}
	return nil, output, nil
}
