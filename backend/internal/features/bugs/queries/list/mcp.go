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
	ReportedBy       string `json:"reported_by,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`

	TakenBy string `json:"taken_by,omitempty" jsonschema:"username de la persona que lo tiene tomado; vacío si nadie"`
	TakenAt string `json:"taken_at,omitempty" jsonschema:"desde cuándo lo tiene tomado (RFC 3339)"`
}

type toolOutput struct {
	Bugs []toolBug `json:"bugs"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_bugs",
		Description: "Lista los bugs de un proyecto de PMT por fecha de creación. Funciona con cualquier status del proyecto. Cada bug trae quién lo tiene tomado (taken_by, taken_at): uno tomado por otra persona solo se lee. Para obtener project_id usa list_projects.",
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
		takenBy, takenAt := platformmcp.HoldFields(item.Hold)
		bugs = append(bugs, toolBug{
			ID:               item.ID.String(),
			ProjectID:        item.ProjectID.String(),
			Title:            item.Title,
			Description:      item.Description,
			StepsToReproduce: item.StepsToReproduce,
			ProductVersion:   item.ProductVersion,
			Status:           string(item.Status),
			ReportedBy:       item.ReportedBy,
			CreatedAt:        item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
			TakenBy:          takenBy,
			TakenAt:          takenAt,
		})
	}
	return nil, toolOutput{Bugs: bugs}, nil
}
