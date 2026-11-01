package changestatusbatch

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string   `json:"stage_id" jsonschema:"UUID de la etapa"`
	TaskIDs []string `json:"task_ids" jsonschema:"UUID de las tareas, todas de esa etapa y sin repetir"`
	Status  string   `json:"status" jsonschema:"Nuevo status de todas: backlog, ready, in_dev, in_review, changes_requested, in_qa, pending_close, done o cancelled"`
	Reason  string   `json:"reason,omitempty" jsonschema:"Motivo del cambio de status; es el mismo para todas y queda en el historial"`
}

type toolTask struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Assignee    string `json:"assignee,omitempty"`
}

type toolOutput struct {
	Tasks []toolTask `json:"tasks"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "change_tasks_status",
		Description: "Cambia el status de varias tareas de una etapa de PMT a un mismo status, todas o ninguna, y devuelve las tareas actualizadas. Cada tarea valida por separado lo mismo que change_task_status y puede venir de otro status; si una falla, ninguna cambia y el error nombra la tarea. Deja un registro inmutable por tarea. Usa list_tasks para obtener los ID. Los bugs no tienen lote.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}
	taskIDs := make([]uuid.UUID, 0, len(input.TaskIDs))
	for _, raw := range input.TaskIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_ids inválido"))
		}
		taskIDs = append(taskIDs, id)
	}

	items, err := h.Handle(ctx, Command{StageID: stageID, TaskIDs: taskIDs, Status: domain.Status(input.Status), Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Tasks: make([]toolTask, 0, len(items))}
	for _, item := range items {
		output.Tasks = append(output.Tasks, toolTask{
			ID:          item.ID.String(),
			StageID:     item.StageID.String(),
			Title:       item.Title,
			Description: item.Description,
			Status:      string(item.Status),
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
			Assignee:    item.Assignee,
		})
	}
	return nil, output, nil
}
