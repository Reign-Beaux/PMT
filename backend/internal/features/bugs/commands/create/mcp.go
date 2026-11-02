package create

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID        string `json:"project_id" jsonschema:"UUID del proyecto, que debe estar en working"`
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
	ReportedBy       string `json:"reported_by,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "create_bug",
		Description: "Registra un bug en un proyecto de PMT. Solo PP y el usuario. No pertenece a ninguna etapa y nace en status reported. Solo funciona si el proyecto está en working. Pide steps_to_reproduce y product_version, la versión donde apareció, antes de llamar a esta herramienta.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		ProjectID:        id,
		Title:            input.Title,
		Description:      input.Description,
		StepsToReproduce: input.StepsToReproduce,
		ProductVersion:   input.ProductVersion,
		Actor:            platformmcp.ActorFrom(req),
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
		ReportedBy:       result.ReportedBy,
		CreatedAt:        result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
