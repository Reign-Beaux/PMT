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
	BugID  string `json:"bug_id" jsonschema:"UUID del bug"`
	Reason string `json:"reason" jsonschema:"Motivo de la liberación; obligatorio, queda en el historial de status"`
}

type toolOutput struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "release_bug",
		Description: "Libera un bug de PMT tomado por una persona: lo deja sin tomar, sin cambiar su status, y registra el motivo en su historial de status (list_bug_status_changes). Solo el owner del proyecto, con perfil User o PP. Quien escriba después no lo toma: solo lo toma quien lo pase a in_dev. Usa list_bugs para ver el ID del bug y quién lo tiene (taken_by).",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	bugID, err := uuid.Parse(input.BugID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("bug_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: bugID, Reason: input.Reason, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}
	return nil, toolOutput{
		ID:        result.ID.String(),
		ProjectID: result.ProjectID.String(),
		Title:     result.Title,
		Status:    string(result.Status),
		UpdatedAt: result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
