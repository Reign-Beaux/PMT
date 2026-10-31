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
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "add_stage",
		Description: "Agrega una etapa nueva al final de un proyecto y la registra como cambio inmutable, con su motivo y quién lo pidió. Es la forma de agregar una etapa cuando el proyecto ya no está en planning. Devuelve la etapa y su versión.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
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
	}, nil
}
