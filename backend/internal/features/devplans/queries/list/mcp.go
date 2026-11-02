package list

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id,omitempty" jsonschema:"UUID de la etapa"`
	TaskID  string `json:"task_id,omitempty" jsonschema:"UUID de una tarea: los planes que la vinculan"`
	BugID   string `json:"bug_id,omitempty" jsonschema:"UUID del bug"`
}

type toolPlan struct {
	ID                 string   `json:"id"`
	StageID            string   `json:"stage_id,omitempty"`
	BugID              string   `json:"bug_id,omitempty"`
	CreatedBy          string   `json:"created_by"`
	Version            int      `json:"version"`
	Scope              string   `json:"scope"`
	TechnicalDecisions string   `json:"technical_decisions"`
	ScreenStates       string   `json:"screen_states"`
	EdgeCases          string   `json:"edge_cases"`
	ErrorHandling      string   `json:"error_handling"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
	Reason             string   `json:"reason,omitempty"`
	VersionCreatedBy   string   `json:"version_created_by"`
	TaskIDs            []string `json:"task_ids"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type toolOutput struct {
	Plans []toolPlan `json:"plans"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_dev_plans",
		Description: "Lista los planes de desarrollo de una etapa, una tarea o un bug, cada uno con su versión más reciente, que es la que vale, y sus tareas. Exige exactamente uno de stage_id, task_id y bug_id. Disponible para todos los perfiles. No comprueba que el elemento exista: sin planes responde lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	var query Query
	var err error
	if query.StageID, err = parseOptional(input.StageID, "stage_id inválido"); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	if query.TaskID, err = parseOptional(input.TaskID, "task_id inválido"); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	if query.BugID, err = parseOptional(input.BugID, "bug_id inválido"); err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	items, err := h.Handle(ctx, query)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Plans: make([]toolPlan, 0, len(items))}
	for _, item := range items {
		taskIDs := make([]string, 0, len(item.TaskIDs))
		for _, id := range item.TaskIDs {
			taskIDs = append(taskIDs, id.String())
		}
		plan := toolPlan{
			ID:                 item.ID.String(),
			CreatedBy:          item.CreatedBy,
			Version:            item.Version,
			Scope:              item.Scope,
			TechnicalDecisions: item.TechnicalDecisions,
			ScreenStates:       item.ScreenStates,
			EdgeCases:          item.EdgeCases,
			ErrorHandling:      item.ErrorHandling,
			AcceptanceCriteria: item.AcceptanceCriteria,
			Reason:             item.Reason,
			VersionCreatedBy:   item.VersionCreatedBy,
			TaskIDs:            taskIDs,
			CreatedAt:          item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          item.UpdatedAt.Format(time.RFC3339),
		}
		if item.StageID != nil {
			plan.StageID = item.StageID.String()
		}
		if item.BugID != nil {
			plan.BugID = item.BugID.String()
		}
		output.Plans = append(output.Plans, plan)
	}
	return nil, output, nil
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
