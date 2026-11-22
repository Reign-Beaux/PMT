package updatemember

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto"`
	Username  string `json:"username" jsonschema:"Username del miembro"`
	Role      string `json:"role" jsonschema:"Rol nuevo: editor o viewer"`
}

type toolOutput struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "update_project_member",
		Description: "Cambia el rol de un miembro de un proyecto de PMT entre editor y viewer. Solo el owner del proyecto, sin perfil declarado (perfil User). El owner no cambia de rol.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}
	item, err := h.Handle(ctx, Command{ProjectID: projectID, Username: input.Username, Role: input.Role, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{Username: item.Username, Name: item.Name, Role: string(item.Role)}, nil
}
