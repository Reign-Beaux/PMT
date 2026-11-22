package importstage

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	projects "pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

func call(t *testing.T, repo Repository, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, kernel.Actor{Name: "saul", Profile: kernel.ProfileUser})
	New(repo).RegisterTool(server)

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "import_stage", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolImportsStagesInWorking(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusWorking}

	result := call(t, repo, map[string]any{
		"project_id": uuid.New().String(), "content": validFile, "reason": "Petición", "requested_by": "PM",
	})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	output := result.StructuredContent.(map[string]any)
	list, ok := output["stages"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("salida = %+v", output)
	}
	first := list[0].(map[string]any)
	if first["stage_name"] != "Pagos" || first["task_count"] != float64(2) || first["stage_id"] != repo.saved[0].Stage.ID.String() {
		t.Errorf("primera etapa = %+v", first)
	}
	second := list[1].(map[string]any)
	if second["stage_name"] != "Reportes" || second["task_count"] != float64(0) {
		t.Errorf("segunda etapa = %+v", second)
	}
}

func TestToolImportsStagesInPlanningWithoutReason(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusPlanning}

	result := call(t, repo, map[string]any{
		"project_id": uuid.New().String(), "content": validFile,
	})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	output := result.StructuredContent.(map[string]any)
	list := output["stages"].([]any)
	if len(list) != 2 || len(repo.saved) != 2 {
		t.Fatalf("salida = %+v, guardadas = %d", output, len(repo.saved))
	}
	for _, plan := range repo.saved {
		if plan.Stage.InOwnPlanning() {
			t.Errorf("etapa = %+v", plan.Stage)
		}
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		content   string
		want      string
	}{
		{"project_id", "x", validFile, "project_id inválido"},
		{"content vacío", uuid.New().String(), "", "el YAML está vacío"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := call(t, &fakeRepo{status: projects.StatusWorking}, map[string]any{
				"project_id": tt.projectID, "content": tt.content, "reason": "M", "requested_by": "PM",
			})

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
		})
	}
}

func TestToolRejectsPathArgument(t *testing.T) {
	repo := &fakeRepo{status: projects.StatusPlanning}

	result := call(t, repo, map[string]any{"project_id": uuid.New().String(), "path": "/tmp/etapas.yaml"})

	if !result.IsError || len(repo.saved) != 0 {
		t.Errorf("resultado = %+v", result)
	}
}
