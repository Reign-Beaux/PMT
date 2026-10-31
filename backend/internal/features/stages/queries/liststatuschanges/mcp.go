package liststatuschanges

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID string `json:"stage_id" jsonschema:"UUID de la etapa; sirve aunque ya no exista"`
}

type toolChange struct {
	ID           string `json:"id"`
	StageID      string `json:"stage_id"`
	FromStatus   string `json:"from_status"`
	ToStatus     string `json:"to_status"`
	ActorName    string `json:"actor_name"`
	ActorProfile string `json:"actor_profile"`
	Reason       string `json:"reason"`
	ChangedAt    string `json:"changed_at"`
}

type toolOutput struct {
	Changes []toolChange `json:"changes"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_stage_status_changes",
		Description: "Lista el historial inmutable de cambios de status de una etapa por orden de fecha: desde, hacia, actor, perfil, fecha y motivo. Funciona aunque la etapa esté retirada o eliminada. Sin cambios devuelve una lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	items, err := h.Handle(ctx, id)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	changes := make([]toolChange, 0, len(items))
	for _, item := range items {
		changes = append(changes, toolChange{
			ID:           item.ID.String(),
			StageID:      item.StageID.String(),
			FromStatus:   string(item.FromStatus),
			ToStatus:     string(item.ToStatus),
			ActorName:    item.ActorName,
			ActorProfile: item.ActorProfile,
			Reason:       item.Reason,
			ChangedAt:    item.ChangedAt.Format(time.RFC3339Nano),
		})
	}
	return nil, toolOutput{Changes: changes}, nil
}
