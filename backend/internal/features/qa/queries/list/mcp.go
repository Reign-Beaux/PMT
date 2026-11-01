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
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
	TaskID  string `json:"task_id,omitempty" jsonschema:"UUID de una tarea: solo los planes que la vinculan"`
}

type toolPlan struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	CreatedBy   string   `json:"created_by"`
	ReviewNotes string   `json:"review_notes,omitempty"`
	ReviewedBy  string   `json:"reviewed_by,omitempty"`
	Version     int      `json:"version"`
	HappyPath   string   `json:"happy_path"`
	ErrorCases  string   `json:"error_cases"`
	Reason      string   `json:"reason,omitempty"`
	TaskIDs     []string `json:"task_ids"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type toolOutput struct {
	Plans []toolPlan `json:"plans"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_test_plans",
		Description: "Lista los planes de pruebas de una etapa, con status, última versión, tareas y última revisión del TL. Con task_id, solo los que vinculan esa tarea. Disponible para todos los perfiles. No comprueba que la etapa exista: sin planes responde lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}
	var taskID *uuid.UUID
	if input.TaskID != "" {
		parsed, err := uuid.Parse(input.TaskID)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
		}
		taskID = &parsed
	}

	items, err := h.Handle(ctx, stageID, taskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{Plans: make([]toolPlan, 0, len(items))}
	for _, item := range items {
		taskIDs := make([]string, 0, len(item.TaskIDs))
		for _, id := range item.TaskIDs {
			taskIDs = append(taskIDs, id.String())
		}
		output.Plans = append(output.Plans, toolPlan{
			ID:          item.ID.String(),
			Title:       item.Title,
			Status:      string(item.Status),
			CreatedBy:   item.CreatedBy,
			ReviewNotes: item.ReviewNotes,
			ReviewedBy:  item.ReviewedBy,
			Version:     item.Version,
			HappyPath:   item.HappyPath,
			ErrorCases:  item.ErrorCases,
			Reason:      item.Reason,
			TaskIDs:     taskIDs,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return nil, output, nil
}
