package add

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID     string `json:"stage_id" jsonschema:"UUID de la etapa a la que pertenece la tarea"`
	Title       string `json:"title" jsonschema:"Título de la tarea"`
	Description string `json:"description" jsonschema:"Descripción de la tarea"`
	Reason      string `json:"reason" jsonschema:"Motivo del cambio"`
	RequestedBy string `json:"requested_by" jsonschema:"Quién pidió el cambio"`
}

type toolOutput struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "add_task",
		Description: "Agrega una tarea nueva al final de una etapa y la registra como cambio inmutable, con su motivo y quién lo pidió. Es la forma de agregar una tarea cuando el proyecto ya no está en planning. Se rechaza si la etapa está en su planning propio: ahí usa create_task. Devuelve la tarea y su versión.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		Actor:       platformmcp.ActorFrom(req),
		StageID:     stageID,
		Title:       input.Title,
		Description: input.Description,
		Reason:      input.Reason,
		RequestedBy: input.RequestedBy,
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:          result.ID.String(),
		StageID:     result.StageID.String(),
		Title:       result.Title,
		Description: result.Description,
		Status:      string(result.Status),
		Version:     result.Version,
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
