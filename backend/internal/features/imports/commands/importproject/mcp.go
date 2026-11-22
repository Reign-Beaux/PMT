package importproject

import (
	"context"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	Content string `json:"content" jsonschema:"Texto completo del YAML con raíz project, hasta 1 MiB"`
}

type toolOutput struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	StageCount  int    `json:"stage_count"`
	TaskCount   int    `json:"task_count"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "import_project",
		Description: "Crea un proyecto nuevo con todas sus etapas y tareas desde el texto de un YAML, que se pasa en content (no una ruta). Solo PP y el usuario. Quien importa queda como owner del proyecto; con un token de agente, el dueño de la instancia. Es todo o nada: si el YAML tiene un error, no se crea nada y el mensaje indica su ubicación. No agrega datos a un proyecto existente. Importar dos veces el mismo YAML crea dos proyectos.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	result, err := h.Handle(ctx, Command{Content: input.Content, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ProjectID:   result.ProjectID.String(),
		ProjectName: result.ProjectName,
		StageCount:  result.StageCount,
		TaskCount:   result.TaskCount,
	}, nil
}
