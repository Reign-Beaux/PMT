package update

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID     string `json:"stage_id" jsonschema:"UUID de la etapa"`
	Name        string `json:"name" jsonschema:"Nuevo nombre de la etapa"`
	Description string `json:"description" jsonschema:"Nueva descripción de la etapa"`
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
		Name:        "update_stage",
		Description: "Reemplaza el nombre y la descripción de una etapa de PMT y devuelve la etapa completa. Solo funciona si la etapa está en su planning propio con el proyecto en working, o si su proyecto está en planning y la etapa no tiene historial; si no, usa revise_stage. Usa list_stages para obtener su ID y sus valores actuales, y envía esos valores en los campos que no cambien.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
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
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),

		AddReason:      result.AddReason,
		AddRequestedBy: result.AddRequestedBy,
	}, nil
}
