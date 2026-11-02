package revise

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	PlanID     string   `json:"plan_id" jsonschema:"UUID del plan"`
	HappyPath  string   `json:"happy_path,omitempty" jsonschema:"Casos del camino feliz de la versión nueva"`
	ErrorCases string   `json:"error_cases,omitempty" jsonschema:"Casos de error de la versión nueva"`
	Reason     string   `json:"reason" jsonschema:"Motivo de la versión nueva"`
	TaskIDs    []string `json:"task_ids" jsonschema:"UUID de las tareas de la etapa; reemplazan a las vinculadas"`
}

type toolOutput struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "revise_test_plan",
		Description: "Crea una versión nueva e inmutable de un plan de pruebas y reemplaza sus tareas. Solo QAP, con motivo obligatorio. Funciona en draft, returned o approved y deja el plan en draft; en submitted se rechaza. El plan aprobado necesita aprobarse otra vez. Usa list_test_plans para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.PlanID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("plan_id inválido"))
	}
	taskIDs := make([]uuid.UUID, 0, len(input.TaskIDs))
	for _, raw := range input.TaskIDs {
		taskID, err := uuid.Parse(raw)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_ids inválido"))
		}
		taskIDs = append(taskIDs, taskID)
	}

	result, err := h.Handle(ctx, Command{ID: id, HappyPath: input.HappyPath, ErrorCases: input.ErrorCases, Reason: input.Reason, TaskIDs: taskIDs, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Status: string(result.Status), Version: result.Version}, nil
}
