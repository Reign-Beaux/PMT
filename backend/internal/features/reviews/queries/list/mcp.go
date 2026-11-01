package list

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
	ItemKind string `json:"item_kind,omitempty" jsonschema:"Tipo de elemento: task o bug. Va con item_id, sin stage_id"`
	ItemID   string `json:"item_id,omitempty" jsonschema:"UUID de la tarea o el bug. Va con item_kind, sin stage_id"`
	StageID  string `json:"stage_id,omitempty" jsonschema:"UUID de la etapa: lista los hallazgos de sus tareas, incluidas las retiradas. Va sin item_kind ni item_id"`
	Pending  bool   `json:"pending,omitempty" jsonschema:"Solo los abiertos, bloqueantes primero"`
}

type toolFinding struct {
	ID           string `json:"id"`
	ItemKind     string `json:"item_kind"`
	ItemID       string `json:"item_id"`
	Description  string `json:"description"`
	Blocking     bool   `json:"blocking"`
	Status       string `json:"status"`
	ActorName    string `json:"actor_name"`
	ActorProfile string `json:"actor_profile"`
	ClosedBy     string `json:"closed_by,omitempty"`
	CloseReason  string `json:"close_reason,omitempty"`
	CreatedAt    string `json:"created_at"`
	ClosedAt     string `json:"closed_at,omitempty"`
}

type toolOutput struct {
	Findings []toolFinding `json:"findings"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_findings",
		Description: "Lista por fecha los hallazgos de una tarea o un bug, con item_kind e item_id, o los de todas las tareas de una etapa, con stage_id; exactamente uno de los dos modos. Cada hallazgo indica su item_kind e item_id. Los bugs no tienen etapa. Con pending, solo los abiertos, bloqueantes primero. Disponible para todos los perfiles. No comprueba que el elemento o la etapa existan: sin hallazgos responde lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	query := Query{ItemKind: domain.ItemKind(input.ItemKind), Pending: input.Pending}
	var err error
	if input.ItemID != "" {
		if query.ItemID, err = uuid.Parse(input.ItemID); err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("item_id inválido"))
		}
	}
	if input.StageID != "" {
		if query.StageID, err = uuid.Parse(input.StageID); err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
		}
	}

	items, err := h.Handle(ctx, query)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Findings: make([]toolFinding, 0, len(items))}
	for _, item := range items {
		finding := toolFinding{
			ID:           item.ID.String(),
			ItemKind:     string(item.ItemKind),
			ItemID:       item.ItemID.String(),
			Description:  item.Description,
			Blocking:     item.Blocking,
			Status:       string(item.Status),
			ActorName:    item.ActorName,
			ActorProfile: string(item.ActorProfile),
			ClosedBy:     item.ClosedBy,
			CloseReason:  item.CloseReason,
			CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		}
		if item.ClosedAt != nil {
			finding.ClosedAt = item.ClosedAt.Format(time.RFC3339)
		}
		output.Findings = append(output.Findings, finding)
	}
	return nil, output, nil
}
