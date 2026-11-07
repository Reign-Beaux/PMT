package revise

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
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "revise_stage",
		Description: "Registra una nueva versión de una etapa como cambio inmutable, con su motivo y quién lo pidió, y devuelve la etapa y su versión. Solo PP, el usuario o el TL de la etapa. Es la forma de editar una etapa cuando el proyecto ya no está en planning o la etapa ya tiene historial. Se rechaza si la etapa está en su planning propio. Usa list_stages para obtener su ID y sus valores actuales, y envía esos valores en los campos que no cambien.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		Actor:       platformmcp.ActorFrom(req),
		ID:          id,
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
	}, nil
}
