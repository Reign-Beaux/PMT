package assign

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	TaskID string `json:"task_id" jsonschema:"UUID de la tarea"`
	Agent  string `json:"agent" jsonschema:"Nombre del dev del equipo de la etapa"`
}

type toolOutput struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Assignee    string `json:"assignee,omitempty"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "assign_task",
		Description: "Pone o cambia el dueño de una tarea ya tomada de PMT. Solo el TL de la etapa. El agente debe estar activo y en el equipo. Se rechaza en backlog, ready, done y cancelled: una tarea libre la toma un dev con change_task_status.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: taskID, Agent: input.Agent, Actor: platformmcp.ActorFrom(req)})
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
		Assignee:    result.Assignee,
	}, nil
}
