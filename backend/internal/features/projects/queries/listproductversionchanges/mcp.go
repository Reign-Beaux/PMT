package listproductversionchanges

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto; sirve aunque ya no exista"`
}

type toolChange struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	FromVersion  string `json:"from_version"`
	ToVersion    string `json:"to_version"`
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
		Name:        "list_product_version_changes",
		Description: "Lista el historial inmutable de cambios de versión de producto de un proyecto en orden cronológico: versión anterior, versión nueva, motivo, actor, perfil y fecha. Lo escribe change_product_version; los cambios hechos en planning con update_project no quedan aquí. Funciona aunque el proyecto ya no exista. Sin cambios devuelve una lista vacía.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	items, err := h.Handle(ctx, id)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	changes := make([]toolChange, 0, len(items))
	for _, item := range items {
		changes = append(changes, toolChange{
			ID:           item.ID.String(),
			ProjectID:    item.ProjectID.String(),
			FromVersion:  item.FromVersion,
			ToVersion:    item.ToVersion,
			ActorName:    item.ActorName,
			ActorProfile: item.ActorProfile,
			Reason:       item.Reason,
			ChangedAt:    item.ChangedAt.Format(time.RFC3339Nano),
		})
	}
	return nil, toolOutput{Changes: changes}, nil
}
