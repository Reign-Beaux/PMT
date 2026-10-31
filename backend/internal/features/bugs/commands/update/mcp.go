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
	BugID            string `json:"bug_id" jsonschema:"UUID del bug"`
	Title            string `json:"title" jsonschema:"Título del bug"`
	Description      string `json:"description" jsonschema:"Descripción del bug; puede ir vacía"`
	StepsToReproduce string `json:"steps_to_reproduce" jsonschema:"Pasos para reproducirlo"`
	ProductVersion   string `json:"product_version" jsonschema:"Versión del producto donde apareció"`
}

type toolOutput struct {
	ID               string `json:"id"`
	ProjectID        string `json:"project_id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	StepsToReproduce string `json:"steps_to_reproduce"`
	ProductVersion   string `json:"product_version"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "update_bug",
		Description: "Reemplaza título, descripción, pasos para reproducirlo y versión de un bug, para corregir el reporte, y devuelve el bug completo. No cambia su status. Solo funciona si el proyecto está en working. Usa list_bugs para obtener su ID y sus valores actuales, y envía esos valores en los campos que no cambien.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.BugID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("bug_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ID:               id,
		Title:            input.Title,
		Description:      input.Description,
		StepsToReproduce: input.StepsToReproduce,
		ProductVersion:   input.ProductVersion,
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:               result.ID.String(),
		ProjectID:        result.ProjectID.String(),
		Title:            result.Title,
		Description:      result.Description,
		StepsToReproduce: result.StepsToReproduce,
		ProductVersion:   result.ProductVersion,
		Status:           string(result.Status),
		CreatedAt:        result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
