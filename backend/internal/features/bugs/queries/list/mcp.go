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

type toolBug struct {
	ID               string `json:"id"`
	ProjectID        string `json:"project_id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	StepsToReproduce string `json:"steps_to_reproduce"`
	ProductVersion   string `json:"product_version"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type toolOutput struct {
	Bugs []toolBug `json:"bugs"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_bugs",
		Description: "Lista los bugs de un proyecto de PMT por fecha de creación. Funciona con cualquier status del proyecto. Para obtener project_id usa list_projects.",
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

	bugs := make([]toolBug, 0, len(items))
	for _, item := range items {
		bugs = append(bugs, toolBug{
			ID:               item.ID.String(),
			ProjectID:        item.ProjectID.String(),
			Title:            item.Title,
			Description:      item.Description,
			StepsToReproduce: item.StepsToReproduce,
			ProductVersion:   item.ProductVersion,
			Status:           string(item.Status),
			CreatedAt:        item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return nil, toolOutput{Bugs: bugs}, nil
}
