package changestatus

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	BugID  string `json:"bug_id" jsonschema:"UUID del bug"`
	Status string `json:"status" jsonschema:"Nuevo status: reported, confirmed, in_progress, in_review, changes_requested, in_qa, done o rejected"`
	Reason string `json:"reason,omitempty" jsonschema:"Motivo del cambio de status; queda en el historial"`
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
		Name:        "change_bug_status",
		Description: "Cambia el status de un bug de PMT y devuelve el bug completo. Deja un registro inmutable, consultable con list_bug_status_changes. Solo admite las transiciones permitidas; cualquier otra, incluida la del status a sí mismo, se rechaza. No depende del status del proyecto. Usa list_bugs para obtener el ID del bug y su status actual.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.BugID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("bug_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Status: domain.Status(input.Status), Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
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
