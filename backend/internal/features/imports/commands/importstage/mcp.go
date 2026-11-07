package importstage

import (
	"context"
	"path/filepath"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID   string `json:"project_id" jsonschema:"UUID del proyecto, que debe estar en working"`
	Path        string `json:"path" jsonschema:"Ruta absoluta del fichero YAML, en la máquina donde corre pmt-server"`
	Reason      string `json:"reason" jsonschema:"Motivo de la etapa"`
	RequestedBy string `json:"requested_by" jsonschema:"Quién pidió la etapa"`
}

type toolOutput struct {
	ProjectID string `json:"project_id"`
	StageID   string `json:"stage_id"`
	StageName string `json:"stage_name"`
	TaskCount int    `json:"task_count"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "import_stage",
		Description: "Crea una etapa con sus tareas desde un fichero YAML con raíz stage, en un proyecto en working. Solo PP y el usuario. La etapa nace en su planning propio, como con add_stage. Es todo o nada: si el fichero tiene un error, no se crea nada y el mensaje indica su ubicación. Importar dos veces el mismo fichero crea dos etapas.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}
	if !filepath.IsAbs(input.Path) {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("path debe ser una ruta absoluta"))
	}

	result, err := h.Handle(ctx, Command{
		Actor:       platformmcp.ActorFrom(req),
		ProjectID:   projectID,
		Path:        input.Path,
		Reason:      input.Reason,
		RequestedBy: input.RequestedBy,
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ProjectID: result.ProjectID.String(),
		StageID:   result.StageID.String(),
		StageName: result.StageName,
		TaskCount: result.TaskCount,
	}, nil
}
