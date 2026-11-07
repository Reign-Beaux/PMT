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
	StageID     string `json:"stage_id" jsonschema:"UUID de la etapa a la que pertenece la tarea"`
	Title       string `json:"title" jsonschema:"Título de la tarea"`
	Description string `json:"description" jsonschema:"Descripción de la tarea"`
}

type toolOutput struct {
	ID          string `json:"id"`
	StageID     string `json:"stage_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "create_task",
		Description: "Crea una tarea en una etapa de PMT. Nace en estado backlog. Solo funciona si la etapa está en su planning propio con el proyecto en working, o si el proyecto de la etapa está en planning y la etapa no nació con add_stage; si no, usa add_task. Si el usuario no indicó la etapa, el título o la descripción, pídeselos antes de llamar a esta herramienta. Para obtener stage_id a partir de nombres usa list_projects y list_stages.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, req *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	stageID, err := uuid.Parse(input.StageID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("stage_id inválido"))
	}

	result, err := h.Handle(ctx, Command{
		Actor:       platformmcp.ActorFrom(req),
		StageID:     stageID,
		Title:       input.Title,
		Description: input.Description,
	})
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	return nil, toolOutput{
		ID:          result.ID.String(),
		StageID:     result.StageID.String(),
		Title:       result.Title,
		Description: result.Description,
		Status:      string(result.Status),
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   result.UpdatedAt.Format(time.RFC3339),
	}, nil
}
