package delete

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	TaskID string `json:"task_id" jsonschema:"UUID de la tarea"`
}

type toolOutput struct {
	DeletedID string `json:"deleted_id"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "delete_task",
		Description: "Elimina una tarea de PMT y devuelve su ID. Solo PP y el usuario. Solo funciona si la etapa está en su planning propio con el proyecto en working, o si el proyecto de su etapa está en planning y ni la tarea tiene historial ni la etapa nació con add_stage; si no, usa retire_task. Usa list_tasks para obtener el ID de la tarea.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
	}

	if err := h.Handle(ctx, id, platformmcp.ActorFrom(req)); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{DeletedID: id.String()}, nil
}
