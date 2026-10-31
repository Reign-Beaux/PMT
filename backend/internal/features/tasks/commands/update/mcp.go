package update

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	TaskID      string `json:"task_id" jsonschema:"UUID de la tarea"`
	Title       string `json:"title" jsonschema:"Nuevo título de la tarea"`
	Description string `json:"description" jsonschema:"Nueva descripción de la tarea"`
}

type toolOutput struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "update_task",
		Description: "Reemplaza el título y la descripción de una tarea de PMT y devuelve la tarea completa. Solo funciona si el proyecto de su etapa está en planning y la tarea no tiene historial; si no, usa revise_task. Usa list_tasks para obtener su ID y sus valores actuales, y envía esos valores en los campos que no cambien.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ID:          id,
		Title:       input.Title,
		Description: input.Description,
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:          result.ID.String(),
		StageID:     result.StageID.String(),
		Title:       result.Title,
		Description: result.Description,
		Status:      string(result.Status),
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
