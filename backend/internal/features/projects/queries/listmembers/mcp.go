package listmembers

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto"`
}

type toolMember struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

type toolOutput struct {
	Members []toolMember `json:"members"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_project_members",
		Description: "Lista los miembros de un proyecto de PMT con su rol: owner, editor o viewer. Cualquier miembro del proyecto, con cualquier perfil.",
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
	members := make([]toolMember, 0, len(items))
	for _, item := range items {
		members = append(members, toolMember{Username: item.Username, Name: item.Name, Role: string(item.Role)})
	}
	return nil, toolOutput{Members: members}, nil
}
