package startstage

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
	Reason  string `json:"reason,omitempty" jsonschema:"Motivo; queda en el historial de status de la etapa y de cada tarea"`
}

type toolTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type toolOutput struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	ProjectID     string     `json:"project_id"`
	ProjectStatus string     `json:"project_status"`
	Tasks         []toolTask `json:"tasks"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "start_stage",
		Description: "Inicia una etapa de PMT en stand_by: la pasa a in_dev, sus tareas en backlog a ready y el proyecto a in_dev si sigue en stand_by, todo o nada. Solo el TL de la etapa. Exige proyecto en stand_by, in_dev o working. Las tareas en otro status no se tocan. Deja un registro inmutable por elemento; el del proyecto lleva al TL como actor y el motivo inicio de la etapa. Usa list_stages para obtener el ID de la etapa.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{
		ID:            result.ID.String(),
		Status:        string(result.Status),
		ProjectID:     result.ProjectID.String(),
		ProjectStatus: string(result.ProjectStatus),
		Tasks:         make([]toolTask, 0, len(result.Tasks)),
	}
	for _, t := range result.Tasks {
		output.Tasks = append(output.Tasks, toolTask{ID: t.ID.String(), Status: string(t.Status)})
	}
	return nil, output, nil
}
