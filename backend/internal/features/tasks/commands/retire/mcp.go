package retire

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	TaskID      string `json:"task_id" jsonschema:"UUID de la tarea"`
	Reason      string `json:"reason" jsonschema:"Motivo del cambio"`
	RequestedBy string `json:"requested_by" jsonschema:"Quién pidió el cambio"`
}

type toolOutput struct {
	RetiredID string `json:"retired_id"`
	Version   int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "retire_task",
		Description: "Retira una tarea de PMT y registra el cambio inmutable, con su motivo y quién lo pidió. No borra nada: deja de aparecer en list_tasks y sigue en list_task_changes. El retiro no se deshace. Se rechaza si su etapa está en su planning propio: ahí usa delete_task. Usa list_tasks para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Reason: input.Reason, RequestedBy: input.RequestedBy})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{RetiredID: result.RetiredID.String(), Version: result.Version}, nil
}
