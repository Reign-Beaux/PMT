package importproject

import (
	"context"
	"path/filepath"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	Path string `json:"path" jsonschema:"Ruta absoluta del fichero YAML, en la máquina donde corre pmt-server"`
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
		Description: "Crea un proyecto nuevo con todas sus etapas y tareas desde un fichero YAML. Es todo o nada: si el fichero tiene un error, no se crea nada y el mensaje indica su ubicación. No agrega datos a un proyecto existente. Importar dos veces el mismo fichero crea dos proyectos.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	if !filepath.IsAbs(input.Path) {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("path debe ser una ruta absoluta"))
	}

	result, err := h.Handle(ctx, Command{Path: input.Path})
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
