package listchanges

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
	TaskID  string `json:"task_id,omitempty" jsonschema:"UUID de una tarea para ver solo su historial"`
}

type toolOriginal struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type toolChange struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
	RequestedBy string `json:"requested_by"`
	ChangedAt   string `json:"changed_at"`
}

type toolHistory struct {
	TaskID   string       `json:"task_id"`
	Original toolOriginal `json:"original"`
	Changes  []toolChange `json:"changes"`
}

type toolOutput struct {
	Histories []toolHistory `json:"histories"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_task_changes",
		Description: "Lista la evolución completa de las tareas con cambios de una etapa, incluidas las retiradas: para cada una, el contenido original y sus cambios en orden de versión. Una tarea sin cambios no aparece. Con task_id devuelve solo la de esa tarea.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	var taskID *uuid.UUID
	if input.TaskID != "" {
		id, err := uuid.Parse(input.TaskID)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
		}
		taskID = &id
	}

	items, err := h.Handle(ctx, stageID, taskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	histories := make([]toolHistory, 0, len(items))
	for _, item := range items {
		changes := make([]toolChange, 0, len(item.Changes))
		for _, c := range item.Changes {
			changes = append(changes, toolChange{
				ID:          c.ID.String(),
				Version:     c.Version,
				Kind:        string(c.Kind),
				Title:       c.Title,
				Description: c.Description,
				Reason:      c.Reason,
				RequestedBy: c.RequestedBy,
				ChangedAt:   c.ChangedAt.Format(time.RFC3339),
			})
		}
		histories = append(histories, toolHistory{
			TaskID: item.TaskID.String(),
			Original: toolOriginal{
				Title:       item.Original.Title,
				Description: item.Original.Description,
				CreatedAt:   item.Original.CreatedAt.Format(time.RFC3339),
			},
			Changes: changes,
		})
	}
	return nil, toolOutput{Histories: histories}, nil
}
