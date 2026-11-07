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
	ProjectID   string `json:"project_id" jsonschema:"UUID del proyecto al que pertenece la etapa"`
	Name        string `json:"name" jsonschema:"Nombre de la etapa"`
	Description string `json:"description" jsonschema:"Descripción de la etapa"`
	Reason      string `json:"reason" jsonschema:"Motivo del cambio"`
	RequestedBy string `json:"requested_by" jsonschema:"Quién pidió el cambio"`
}

type toolOutput struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`

	AddReason      string `json:"add_reason,omitempty"`
	AddRequestedBy string `json:"add_requested_by,omitempty"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "add_stage",
		Description: "Agrega una etapa nueva al final de un proyecto cuando ya no está en planning, con su motivo y quién la pidió. Solo PP y el usuario. Con el proyecto en working nace en su planning propio: se define con update_stage y create_task, devuelve version 0 y add_reason, y su registro inmutable se escribe al sacarla de planning con change_stage_status. En otro status se registra de inmediato como cambio, con version 1.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		Actor:       platformmcp.ActorFrom(req),
		ProjectID:   projectID,
		Name:        input.Name,
		Description: input.Description,
		Reason:      input.Reason,
		RequestedBy: input.RequestedBy,
	})
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
		Version:     result.Version,
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),

		AddReason:      result.AddReason,
		AddRequestedBy: result.AddRequestedBy,
	}, nil
}
