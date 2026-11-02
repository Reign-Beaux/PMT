package revise

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	PlanID             string   `json:"plan_id" jsonschema:"UUID del plan"`
	TaskIDs            []string `json:"task_ids,omitempty" jsonschema:"UUID de las tareas de la etapa; reemplazan a las vinculadas. Un plan de bug no lleva tareas"`
	Scope              string   `json:"scope" jsonschema:"Archivos a crear o tocar y lo que queda fuera"`
	TechnicalDecisions string   `json:"technical_decisions" jsonschema:"Estructura, contrato y bibliotecas"`
	ScreenStates       string   `json:"screen_states" jsonschema:"Estados de pantalla: carga, vacío, error y móvil. Si no aplica, escribe el motivo"`
	EdgeCases          string   `json:"edge_cases" jsonschema:"Valores extremos, caracteres especiales y datos corruptos"`
	ErrorHandling      string   `json:"error_handling" jsonschema:"Qué falla con mensaje, nunca en silencio"`
	AcceptanceCriteria string   `json:"acceptance_criteria" jsonschema:"Cada criterio con el comando o la pantalla que lo comprueba contra la API real"`
	Reason             string   `json:"reason" jsonschema:"Motivo de la versión nueva"`
}

type toolOutput struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "revise_dev_plan",
		Description: "Añade una versión nueva e inmutable a un plan de desarrollo y reemplaza sus tareas; las versiones anteriores se conservan. Solo DP, con motivo obligatorio. Lleva completo todo el contenido, y vale la versión más reciente. Usa list_dev_plans para obtener su ID.",
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

	result, err := h.Handle(ctx, Command{
		ID:                 id,
		TaskIDs:            taskIDs,
		Scope:              input.Scope,
		TechnicalDecisions: input.TechnicalDecisions,
		ScreenStates:       input.ScreenStates,
		EdgeCases:          input.EdgeCases,
		ErrorHandling:      input.ErrorHandling,
		AcceptanceCriteria: input.AcceptanceCriteria,
		Reason:             input.Reason,
		Actor:              platformmcp.ActorFrom(req),
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Version: result.Version}, nil
}
