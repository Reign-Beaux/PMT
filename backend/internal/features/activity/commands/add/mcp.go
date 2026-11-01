package add

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
	Body     string `json:"body" jsonschema:"Texto del comentario: la decisión o el bloqueo, sin copiar avisos"`
}

type toolOutput struct {
	ID           string `json:"id"`
	ItemKind     string `json:"item_kind"`
	ItemID       string `json:"item_id"`
	Body         string `json:"body"`
	ActorName    string `json:"actor_name"`
	ActorProfile string `json:"actor_profile"`
	CreatedAt    string `json:"created_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "add_comment",
		Description: "Registra un comentario sobre un proyecto, una etapa, una tarea o un bug: decisiones que no son una transición y bloqueos, sin copiar los avisos. Disponible para todos los perfiles. El elemento debe existir, sin retirar. No cambia su status, y el comentario no se edita ni se borra.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	itemID, err := uuid.Parse(input.ItemID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("item_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ItemKind: domain.ItemKind(input.ItemKind),
		ItemID:   itemID,
		Body:     input.Body,
		Actor:    platformmcp.ActorFrom(req),
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:           result.ID.String(),
		ItemKind:     string(result.ItemKind),
		ItemID:       result.ItemID.String(),
		Body:         result.Body,
		ActorName:    result.ActorName,
		ActorProfile: string(result.ActorProfile),
		CreatedAt:    result.CreatedAt.Format(time.RFC3339),
	}, nil
}
