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
	ProjectID   string `json:"project_id" jsonschema:"UUID del proyecto, en planning o en working"`
	Path        string `json:"path" jsonschema:"Ruta absoluta del fichero YAML, en la máquina donde corre pmt-server"`
	Reason      string `json:"reason,omitempty" jsonschema:"Motivo de las etapas, obligatorio en working"`
	RequestedBy string `json:"requested_by,omitempty" jsonschema:"Quién pidió las etapas, obligatorio en working"`
}

type stageOutput struct {
	StageID   string `json:"stage_id"`
	StageName string `json:"stage_name"`
	TaskCount int    `json:"task_count"`
}

type toolOutput struct {
	ProjectID string        `json:"project_id"`
	Stages    []stageOutput `json:"stages"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "import_stage",
		Description: "Crea una o varias etapas con sus tareas desde un fichero YAML con raíz stages, en un proyecto en planning o en working. Solo PP y el usuario. En planning las crea como con create_stage y en working en su planning propio, como con add_stage, con reason y requested_by obligatorios. Es todo o nada: si el fichero tiene un error, no se crea nada y el mensaje indica su ubicación. Importar dos veces el mismo fichero crea las etapas dos veces.",
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

	output := toolOutput{ProjectID: result.ProjectID.String(), Stages: make([]stageOutput, 0, len(result.Stages))}
	for _, stage := range result.Stages {
		output.Stages = append(output.Stages, stageOutput{
			StageID:   stage.StageID.String(),
			StageName: stage.StageName,
			TaskCount: stage.TaskCount,
		})
	}
	return nil, output, nil
}
