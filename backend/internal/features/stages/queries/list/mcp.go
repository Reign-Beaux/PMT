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
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto"`
}

type toolStage struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type toolOutput struct {
	Stages []toolStage `json:"stages"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_stages",
		Description: "Lista las etapas de un proyecto de PMT, ordenadas por posición. Úsala para obtener el ID de una etapa a partir de su nombre, por ejemplo antes de usar create_task o list_tasks. Para obtener project_id usa list_projects.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	items, err := h.Handle(ctx, projectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	stages := make([]toolStage, 0, len(items))
	for _, item := range items {
		stages = append(stages, toolStage{
			ID:          item.ID.String(),
			ProjectID:   item.ProjectID.String(),
			Name:        item.Name,
			Description: item.Description,
			Position:    item.Position,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return nil, toolOutput{Stages: stages}, nil
}
