package listchanges

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	ProjectID string `json:"project_id" jsonschema:"UUID del proyecto"`
	StageID   string `json:"stage_id,omitempty" jsonschema:"UUID de una etapa para ver solo su historial"`
}

type toolOriginal struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type toolChange struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
	RequestedBy string `json:"requested_by"`
	ChangedAt   string `json:"changed_at"`
}

type toolHistory struct {
	StageID  string       `json:"stage_id"`
	Original toolOriginal `json:"original"`
	Changes  []toolChange `json:"changes"`
}

type toolOutput struct {
	Histories []toolHistory `json:"histories"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "list_stage_changes",
		Description: "Lista la evolución completa de las etapas con cambios de un proyecto, incluidas las retiradas: para cada una, el contenido original y sus cambios en orden de versión. Una etapa sin cambios no aparece. Con stage_id devuelve solo la de esa etapa.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	projectID, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	var stageID *uuid.UUID
	if input.StageID != "" {
		id, err := uuid.Parse(input.StageID)
		if err != nil {
			return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
		}
		stageID = &id
	}

	items, err := h.Handle(ctx, projectID, stageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	histories := make([]toolHistory, 0, len(items))
	for _, item := range items {
		changes := make([]toolChange, 0, len(item.Changes))
		for _, c := range item.Changes {
			changes = append(changes, toolChange{
				ID:          c.ID.String(),
				Version:     c.Version,
				Kind:        string(c.Kind),
				Name:        c.Name,
				Description: c.Description,
				Reason:      c.Reason,
				RequestedBy: c.RequestedBy,
				ChangedAt:   c.ChangedAt.Format(time.RFC3339),
			})
		}
		histories = append(histories, toolHistory{
			StageID: item.StageID.String(),
			Original: toolOriginal{
				Name:        item.Original.Name,
				Description: item.Original.Description,
				CreatedAt:   item.Original.CreatedAt.Format(time.RFC3339),
			},
			Changes: changes,
		})
	}
	return nil, toolOutput{Histories: histories}, nil
}
