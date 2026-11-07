package create

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID            string   `json:"stage_id,omitempty" jsonschema:"UUID de la etapa. Exige task_ids; excluye bug_id"`
	BugID              string   `json:"bug_id,omitempty" jsonschema:"UUID del bug. Excluye stage_id y task_ids"`
	TaskIDs            []string `json:"task_ids,omitempty" jsonschema:"UUID de las tareas de la etapa que cubre el plan"`
	Scope              string   `json:"scope" jsonschema:"Archivos a crear o tocar y lo que queda fuera"`
	TechnicalDecisions string   `json:"technical_decisions" jsonschema:"Estructura, contrato y bibliotecas"`
	ScreenStates       string   `json:"screen_states" jsonschema:"Estados de pantalla: carga, vacío, error y móvil. Si no aplica, escribe el motivo"`
	EdgeCases          string   `json:"edge_cases" jsonschema:"Valores extremos, caracteres especiales y datos corruptos"`
	ErrorHandling      string   `json:"error_handling" jsonschema:"Qué falla con mensaje, nunca en silencio"`
	AcceptanceCriteria string   `json:"acceptance_criteria" jsonschema:"Cada criterio con la tarea a la que pertenece, quién lo comprueba y cómo: Dev con pruebas unitarias, build y lint; QA en pantalla, contra la API real solo si la tarea la consume, y en los datos guardados"`
}

type toolOutput struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "create_dev_plan",
		Description: "Crea un plan de desarrollo, versión 1. Solo DP. Pertenece a una etapa, con stage_id y al menos una tarea de ella sin repetir y sin retirar, o a un bug, con bug_id y sin tareas: exactamente uno de los dos. Todos los campos de contenido son obligatorios; si uno no aplica, se escribe el motivo. No tiene status ni aprobación y no frena transiciones. Usa list_stages, list_tasks y list_bugs para obtener los ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := parseOptional(input.StageID, "stage_id inválido")
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	bugID, err := parseOptional(input.BugID, "bug_id inválido")
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	taskIDs := make([]uuid.UUID, 0, len(input.TaskIDs))
	for _, raw := range input.TaskIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_ids inválido"))
		}
		taskIDs = append(taskIDs, id)
	}

	result, err := h.Handle(ctx, Command{
		StageID:            stageID,
		BugID:              bugID,
		TaskIDs:            taskIDs,
		Scope:              input.Scope,
		TechnicalDecisions: input.TechnicalDecisions,
		ScreenStates:       input.ScreenStates,
		EdgeCases:          input.EdgeCases,
		ErrorHandling:      input.ErrorHandling,
		AcceptanceCriteria: input.AcceptanceCriteria,
		Actor:              platformmcp.ActorFrom(req),
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{ID: result.ID.String(), Version: result.Version}, nil
}

func parseOptional(raw, message string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, kernel.Invalid(message)
	}
	return &id, nil
}
