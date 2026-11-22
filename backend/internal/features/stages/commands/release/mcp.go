package release

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa"`
	Reason  string `json:"reason" jsonschema:"Motivo de la liberación; obligatorio, queda en el historial de status"`
}

type toolOutput struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "release_stage",
		Description: "Libera una etapa de PMT tomada por una persona: la deja sin tomar, sin cambiar su status, y registra el motivo en su historial de status (list_stage_status_changes). Solo el owner del proyecto, con perfil User o PP. Quien escriba después no la toma: solo la toman start_stage o pasarla de nuevo a in_dev. Usa list_stages para ver el ID de la etapa y quién la tiene (taken_by).",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: stageID, Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{
		ID:        result.ID.String(),
		ProjectID: result.ProjectID.String(),
		Name:      result.Name,
		Status:    string(result.Status),
		UpdatedAt: result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
