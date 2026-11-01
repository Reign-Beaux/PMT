package retire

import (
	"context"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

type toolInput struct {
	StageID     string `json:"stage_id" jsonschema:"UUID de la etapa"`
	Reason      string `json:"reason" jsonschema:"Motivo del cambio"`
	RequestedBy string `json:"requested_by" jsonschema:"Quién pidió el cambio"`
}

type toolOutput struct {
	RetiredID string `json:"retired_id"`
	Version   int    `json:"version"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "retire_stage",
		Description: "Retira una etapa de PMT y registra el cambio inmutable, con su motivo y quién lo pidió. No borra nada: deja de aparecer en list_stages y sigue en list_stage_changes. El retiro no se deshace. Solo funciona si la etapa no tiene tareas activas: primero retira sus tareas. Se rechaza si la etapa está en su planning propio. Al retirar la última etapa abierta de la versión 1, las que no tienen registro add, el proyecto pasa solo de in_dev a working si al menos una quedó en done y las demás en done, cancelled o retiradas; retirar una agregada no lo mueve. Usa list_stages para obtener su ID.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{ID: id, Reason: input.Reason, RequestedBy: input.RequestedBy, Actor: platformmcp.ActorFrom(req)})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{RetiredID: result.RetiredID.String(), Version: result.Version}, nil
}
