package changeproductversion

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
	ProductVersion string `json:"product_version" jsonschema:"Nueva versión del producto con la forma X.Y.Z, mayor que la actual"`
	Reason         string `json:"reason" jsonschema:"Motivo del cambio de versión; obligatorio, queda en el historial"`
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
		Name:        "change_product_version",
		Description: "Sube la versión de producto de un proyecto de PMT en working y devuelve el proyecto completo. Solo PP y el usuario. Solo con el proyecto en working; en planning la versión se cambia con update_project. La versión debe tener la forma X.Y.Z, tres enteros sin signo y sin ceros a la izquierda, y ser mayor que la actual si esta también tiene esa forma. El motivo es obligatorio. Deja un registro inmutable, consultable con list_product_version_changes. Usa list_projects para obtener el ID del proyecto y su versión actual.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ID:             id,
		ProductVersion: input.ProductVersion,
		Reason:         input.Reason,
		Actor:          platformmcp.ActorFrom(req),
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
