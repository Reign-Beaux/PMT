package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
	Status  string `json:"status" jsonschema:"Nuevo status: planning, stand_by, in_dev, done, stopped o cancelled"`
	Reason  string `json:"reason,omitempty" jsonschema:"Motivo del cambio de status; queda en el historial"`
}

type toolOutput struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`

	AddReason      string `json:"add_reason,omitempty"`
	AddRequestedBy string `json:"add_requested_by,omitempty"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "change_stage_status",
		Description: "Cambia el status de una etapa de PMT. Deja un registro inmutable, consultable con list_stage_status_changes. Solo admite las transiciones permitidas; cualquier otra, incluida la del status a sí mismo, se rechaza. Depende del proyecto y de las tareas: con el proyecto en stopped o cancelled solo se detiene o se cancela; pasar a in_dev exige proyecto en in_dev o working, y planning → stand_by, al menos una tarea activa. Una etapa solo pasa a done si todas sus tareas están en done. Salir de planning cierra el planning propio de una etapa: escribe su registro add y desde entonces solo admite add_*, revise_* y retire_*. Usa list_stages para obtener el ID de la etapa y su status actual.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: stageID, Status: domain.Status(input.Status), Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:          result.ID.String(),
		ProjectID:   result.ProjectID.String(),
		Name:        result.Name,
		Description: result.Description,
		Position:    result.Position,
		Status:      string(result.Status),
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),

		AddReason:      result.AddReason,
		AddRequestedBy: result.AddRequestedBy,
	}, nil
}
