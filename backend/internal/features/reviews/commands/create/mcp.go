package create

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
	ItemKind    string `json:"item_kind" jsonschema:"Tipo de elemento: task o bug"`
	ItemID      string `json:"item_id" jsonschema:"UUID de la tarea o el bug"`
	Description string `json:"description" jsonschema:"Qué se encontró"`
	Blocking    bool   `json:"blocking" jsonschema:"Si impide aprobar: una falla bloqueante se devuelve de inmediato"`
}

type toolOutput struct {
	ID           string `json:"id"`
	ItemKind     string `json:"item_kind"`
	ItemID       string `json:"item_id"`
	Description  string `json:"description"`
	Blocking     bool   `json:"blocking"`
	Status       string `json:"status"`
	ActorName    string `json:"actor_name"`
	ActorProfile string `json:"actor_profile"`
	CreatedAt    string `json:"created_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "create_finding",
		Description: "Registra un hallazgo de CR o QA sobre una tarea o un bug. Solo CR y QA. No cambia el status del elemento: para devolverlo usa change_tasks_status o change_bug_status. Usa list_tasks o list_bugs para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	itemID, err := uuid.Parse(input.ItemID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("item_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ItemKind:    domain.ItemKind(input.ItemKind),
		ItemID:      itemID,
		Description: input.Description,
		Blocking:    &input.Blocking,
		Actor:       platformmcp.ActorFrom(req),
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:           result.ID.String(),
		ItemKind:     string(result.ItemKind),
		ItemID:       result.ItemID.String(),
		Description:  result.Description,
		Blocking:     result.Blocking,
		Status:       string(result.Status),
		ActorName:    result.ActorName,
		ActorProfile: string(result.ActorProfile),
		CreatedAt:    result.CreatedAt.Format(time.RFC3339),
	}, nil
}
