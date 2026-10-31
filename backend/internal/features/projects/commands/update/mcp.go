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
	ProjectID      string `json:"project_id" jsonschema:"UUID del proyecto"`
	Name           string `json:"name" jsonschema:"Nuevo nombre del proyecto"`
	Description    string `json:"description" jsonschema:"Nueva descripción del proyecto"`
	ProductVersion string `json:"product_version" jsonschema:"Nueva versión del producto; vacía si no tiene"`
}

type toolOutput struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	ProductVersion string `json:"product_version"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "update_project",
		Description: "Reemplaza el nombre, la descripción y la versión de producto de un proyecto de PMT y devuelve el proyecto completo. Fuera de planning solo cambian el nombre y la descripción: la versión de producto debe enviarse igual a la actual. Usa list_projects para obtener su ID y sus valores actuales, y envía esos valores en los campos que no cambien.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ID:             id,
		Name:           input.Name,
		Description:    input.Description,
		ProductVersion: input.ProductVersion,
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:             result.ID.String(),
		Name:           result.Name,
		Description:    result.Description,
		Status:         string(result.Status),
		ProductVersion: result.ProductVersion,
		CreatedAt:      result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
