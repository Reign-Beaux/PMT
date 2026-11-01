package progress

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
}

type toolProject struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	LastChangeAt string `json:"last_change_at"`
}

type toolTask struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Assignee  string `json:"assignee,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

type toolStage struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Lead    string         `json:"lead,omitempty"`
	Members []string       `json:"members"`
	Counts  map[string]int `json:"counts"`
	Tasks   []toolTask     `json:"tasks"`
}

type toolSignal struct {
	Code      string `json:"code"`
	StageID   string `json:"stage_id"`
	StageName string `json:"stage_name"`
}

type toolOutput struct {
	Project toolProject  `json:"project"`
	Stages  []toolStage  `json:"stages"`
	Signals []toolSignal `json:"signals"`
}

// RegisterTool registra la herramienta MCP de la operación.
func (h *Handler) RegisterTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        "get_project_progress",
		Description: "Muestra el avance de un proyecto de PMT en una sola llamada: sus etapas por posición, con TL, equipo, conteo por status y tareas, y las señales de un posible atasco. Solo lectura, para todos los perfiles. Lo retirado no aparece. Señales: ready_without_work, etapa en in_dev con tareas en ready y ninguna en curso; stage_not_closed, etapa en in_dev con todas sus tareas en done o cancelled y al menos una en done; stage_not_started, proyecto en stand_by, in_dev o working, ninguna etapa en in_dev y alguna en stand_by, y nombra la primera por posición. Con el proyecto en stopped o cancelled no se emite ready_without_work ni stage_not_closed. Son hechos, sin umbrales de tiempo. Usa list_projects para obtener el ID del proyecto.",
	}, h.callTool)
}

func (h *Handler) callTool(ctx context.Context, _ *sdk.CallToolRequest, input toolInput) (*sdk.CallToolResult, toolOutput, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(kernel.Invalid("project_id inválido"))
	}

	result, err := h.Handle(ctx, id)
	if err != nil {
		return nil, toolOutput{}, platformmcp.ToolError(err)
	}

	output := toolOutput{
		Project: toolProject{
			ID:           result.Project.ID.String(),
			Name:         result.Project.Name,
			Status:       string(result.Project.Status),
			LastChangeAt: result.Project.LastChangeAt.Format(time.RFC3339),
		},
		Stages:  make([]toolStage, 0, len(result.Stages)),
		Signals: make([]toolSignal, 0, len(result.Signals)),
	}
	for _, s := range result.Stages {
		stage := toolStage{
			ID: s.ID.String(), Name: s.Name, Status: string(s.Status), Lead: s.Lead,
			Members: s.Members, Counts: s.Counts,
			Tasks: make([]toolTask, 0, len(s.Tasks)),
		}
		for _, t := range s.Tasks {
			stage.Tasks = append(stage.Tasks, toolTask{
				ID: t.ID.String(), Title: t.Title, Status: string(t.Status), Assignee: t.Assignee,
				UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
			})
		}
		output.Stages = append(output.Stages, stage)
	}
	for _, s := range result.Signals {
		output.Signals = append(output.Signals, toolSignal{Code: s.Code, StageID: s.StageID.String(), StageName: s.StageName})
	}
	return nil, output, nil
}
