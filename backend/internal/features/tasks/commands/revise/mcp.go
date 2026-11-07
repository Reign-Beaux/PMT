package revise

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
	Reason      string `json:"reason" jsonschema:"Motivo del cambio"`
	RequestedBy string `json:"requested_by" jsonschema:"Quién pidió el cambio"`
}

type toolOutput struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "revise_task",
		Description: "Registra una nueva versión de una tarea como cambio inmutable, con su motivo y quién lo pidió, y devuelve la tarea y su versión. Solo PP y el usuario. Es la forma de editar una tarea cuando el proyecto ya no está en planning o la tarea ya tiene historial. Una tarea en in_review, in_qa o pending_close vuelve a changes_requested, conserva su dueño y deja un registro de status; las demás no cambian de status, y una en done se rechaza: usa add_task. Se rechaza si su etapa está en su planning propio: ahí usa update_task. Usa list_tasks para obtener su ID y sus valores actuales, y envía esos valores en los campos que no cambien.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.TaskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ID:          id,
		Title:       input.Title,
		Description: input.Description,
		Reason:      input.Reason,
		RequestedBy: input.RequestedBy,
		Actor:       platformmcp.ActorFrom(req),
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
		Version:     result.Version,
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
