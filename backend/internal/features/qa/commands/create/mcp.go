package create

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID    string   `json:"stage_id" jsonschema:"UUID de la etapa"`
	Title      string   `json:"title" jsonschema:"Título del plan"`
	HappyPath  string   `json:"happy_path,omitempty" jsonschema:"Casos del camino feliz. Obligatorio para enviar el plan"`
	ErrorCases string   `json:"error_cases,omitempty" jsonschema:"Casos de error. Obligatorio para enviar el plan"`
	TaskIDs    []string `json:"task_ids" jsonschema:"UUID de las tareas de la etapa que cubre el plan"`
}

type toolOutput struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "create_test_plan",
		Description: "Crea un plan de pruebas en draft, versión 1, vinculado a tareas de una etapa. Solo QAP. Exige al menos una tarea, sin repetir, de esa etapa y sin retirar. happy_path y error_cases pueden quedar vacíos hasta enviarlo. Usa list_stages y list_tasks para obtener los ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}
	taskIDs := make([]uuid.UUID, 0, len(input.TaskIDs))
	for _, raw := range input.TaskIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_ids inválido"))
		}
		taskIDs = append(taskIDs, id)
	}

	result, err := h.Handle(ctx, Command{StageID: stageID, Title: input.Title, HappyPath: input.HappyPath, ErrorCases: input.ErrorCases, TaskIDs: taskIDs, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Status: string(result.Status), Version: result.Version}, nil
}
