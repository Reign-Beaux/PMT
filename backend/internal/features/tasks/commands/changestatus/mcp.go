package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	TaskID string `json:"task_id" jsonschema:"UUID de la tarea"`
	Status string `json:"status" jsonschema:"Nuevo status: backlog, ready, in_progress, in_review, changes_requested, in_qa, pending_close, done o cancelled"`
	Reason string `json:"reason,omitempty" jsonschema:"Motivo del cambio de status; queda en el historial"`
}

type toolOutput struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Assignee    string `json:"assignee,omitempty"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "change_task_status",
		Description: "Cambia el status de una tarea de PMT y devuelve la tarea con su última versión. Solo admite las transiciones permitidas; cualquier otra, incluida la del status a sí mismo, se rechaza. Cada transición exige un perfil: el TL debe ser el de la etapa, y solo un dev del equipo toma una tarea en ready, que queda a su nombre y solo él mueve. Cancelar y reabrir exigen reason. No depende del status del proyecto ni de la etapa y no registra un cambio de contenido; deja un registro inmutable, consultable con list_task_status_changes. Usa list_tasks para obtener el ID de la tarea y su status actual.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("task_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: taskID, Status: domain.Status(input.Status), Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:          result.ID.String(),
		StageID:     result.StageID.String(),
		Title:       result.Title,
		Description: result.Description,
		Status:      string(result.Status),
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),
		Assignee:    result.Assignee,
	}, nil
}
