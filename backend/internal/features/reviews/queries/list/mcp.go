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
	ItemKind string `json:"item_kind" jsonschema:"Tipo de elemento: task o bug"`
	ItemID   string `json:"item_id" jsonschema:"UUID de la tarea o el bug"`
	Pending  bool   `json:"pending,omitempty" jsonschema:"Solo los abiertos, bloqueantes primero"`
}

type toolFinding struct {
	ID           string `json:"id"`
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
		Description: "Lista los hallazgos de una tarea o un bug por fecha. Con pending, solo los abiertos, bloqueantes primero. Disponible para todos los perfiles. No comprueba que el elemento exista: sin hallazgos responde lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	itemID, err := uuid.Parse(input.ItemID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("item_id inválido"))
	}

	items, err := h.Handle(ctx, domain.ItemKind(input.ItemKind), itemID, input.Pending)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Findings: make([]toolFinding, 0, len(items))}
	for _, item := range items {
		finding := toolFinding{
			ID:           item.ID.String(),
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
