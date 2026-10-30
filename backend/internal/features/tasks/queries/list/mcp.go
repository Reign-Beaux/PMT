package list

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
}

type toolTask struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Blocked     bool   `json:"blocked"`
	Version     int64  `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type toolOutput struct {
	Tasks []toolTask `json:"tasks"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_tasks",
		Description: "Lista las tareas de una etapa de PMT, de la más antigua a la más reciente. Para obtener stage_id a partir de nombres usa list_projects y list_stages.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	items, err := h.Handle(ctx, stageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	tasks := make([]toolTask, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, toolTask{
			ID:          item.ID.String(),
			StageID:     item.StageID.String(),
			Title:       item.Title,
			Description: item.Description,
			Status:      string(item.Status),
			Blocked:     item.Blocked,
			Version:     item.Version,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return nil, toolOutput{Tasks: tasks}, nil
}
