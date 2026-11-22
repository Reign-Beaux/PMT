package list

import (
	"context"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct{}

type toolProject struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	ProductVersion string `json:"product_version"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type toolOutput struct {
	Projects []toolProject `json:"projects"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_projects",
		Description: "Lista los proyectos de PMT que ves: los que creaste y los que te compartieron como editor (los que te compartieron como viewer solo se consultan en el sitio web); con un token de agente, los del dueño de la instancia. Todos los perfiles. Úsala para obtener el ID de un proyecto a partir de su nombre, por ejemplo antes de usar create_stage.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, _ toolInput) (*sdk.CallToolResult, toolOutput, error) {
	items, err := h.Handle(ctx, platformmcp.ActorFrom(req))
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	projects := make([]toolProject, 0, len(items))
	for _, item := range items {
		projects = append(projects, toolProject{
			ID:             item.ID.String(),
			Name:           item.Name,
			Description:    item.Description,
			Status:         string(item.Status),
			ProductVersion: item.ProductVersion,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return nil, toolOutput{Projects: projects}, nil
}
