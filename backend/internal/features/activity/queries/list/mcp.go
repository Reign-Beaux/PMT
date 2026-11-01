package list

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/activity/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ItemKind string `json:"item_kind" jsonschema:"Tipo de elemento: project, stage, task o bug"`
	ItemID   string `json:"item_id" jsonschema:"UUID del proyecto, la etapa, la tarea o el bug"`
}

type toolComment struct {
	ID           string `json:"id"`
	ItemKind     string `json:"item_kind"`
	ItemID       string `json:"item_id"`
	Body         string `json:"body"`
	ActorName    string `json:"actor_name"`
	ActorProfile string `json:"actor_profile"`
	CreatedAt    string `json:"created_at"`
}

type toolOutput struct {
	Comments []toolComment `json:"comments"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_comments",
		Description: "Lista por fecha los comentarios de un proyecto, una etapa, una tarea o un bug. Disponible para todos los perfiles. No comprueba que el elemento exista: sin comentarios responde lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	itemID, err := uuid.Parse(input.ItemID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("item_id inválido"))
	}

	items, err := h.Handle(ctx, Query{ItemKind: domain.ItemKind(input.ItemKind), ItemID: itemID})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Comments: make([]toolComment, 0, len(items))}
	for _, item := range items {
		output.Comments = append(output.Comments, toolComment{
			ID:           item.ID.String(),
			ItemKind:     string(item.ItemKind),
			ItemID:       item.ItemID.String(),
			Body:         item.Body,
			ActorName:    item.ActorName,
			ActorProfile: string(item.ActorProfile),
			CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		})
	}
	return nil, output, nil
}
