package releaseproject

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto"`
	Reason    string `json:"reason,omitempty" jsonschema:"Motivo; queda en el historial de status del proyecto y de cada etapa"`
}

type toolStage struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type toolOutput struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Stages []toolStage `json:"stages"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "release_project",
		Description: "Libera un proyecto de PMT en planning: lo pasa a stand_by junto con sus etapas en planning, todo o nada, y devuelve las etapas que movió. Solo PP y el usuario. Exige al menos una etapa activa, y cada etapa en planning necesita una tarea activa; si una falla, nada cambia y el error la nombra. Las etapas en otro status no se tocan y las tareas siguen en backlog. Deja un registro inmutable por elemento. Una etapa en planning propio sale con su registro add. Usa list_projects para obtener el ID del proyecto.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{ID: result.ID.String(), Status: string(result.Status), Stages: make([]toolStage, 0, len(result.Stages))}
	for _, s := range result.Stages {
		output.Stages = append(output.Stages, toolStage{ID: s.ID.String(), Status: string(s.Status)})
	}
	return nil, output, nil
}
