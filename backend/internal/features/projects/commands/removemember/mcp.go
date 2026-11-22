package removemember

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
}

type toolOutput struct {
	Removed string `json:"removed"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "remove_project_member",
		Description: "Quita a un miembro de un proyecto de PMT: deja de verlo. Solo el owner del proyecto, sin perfil declarado (perfil User). El owner no se quita.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}
	if err := h.Handle(ctx, Command{ProjectID: projectID, Username: input.Username, Actor: platformmcp.ActorFrom(req)}); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{Removed: input.Username}, nil
}
