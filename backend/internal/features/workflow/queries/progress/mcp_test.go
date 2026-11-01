package progress

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	projects "pmt/internal/features/projects/domain"
	stages "pmt/internal/features/stages/domain"
	tasks "pmt/internal/features/tasks/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

func callTool(t *testing.T, r Reader, projectID string) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(r).RegisterTool(server)

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

	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "get_project_progress",
		Arguments: map[string]any{"project_id": projectID},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolReturnsProgress(t *testing.T) {
	r := reader(projects.StatusInDev, stageFixture{"A", stages.StatusInDev, []tasks.Status{tasks.StatusDone, tasks.StatusReady}})

	result := callTool(t, r, r.project.ID.String())

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output toolOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatal(err)
	}
	if output.Project.Status != "in_dev" || output.Project.LastChangeAt == "" {
		t.Errorf("proyecto = %+v", output.Project)
	}
	if len(output.Stages) != 1 || output.Stages[0].Counts["ready"] != 1 || len(output.Stages[0].Tasks) != 2 {
		t.Errorf("etapas = %+v", output.Stages)
	}
	if len(output.Signals) != 1 || output.Signals[0].Code != SignalReadyWithoutWork || output.Signals[0].StageID != r.stages[0].ID.String() {
		t.Errorf("señales = %+v", output.Signals)
	}
}

func TestToolReturnsEmptyLists(t *testing.T) {
	r := reader(projects.StatusPlanning, stageFixture{"A", stages.StatusPlanning, nil})

	result := callTool(t, r, r.project.ID.String())

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"signals":[]`, `"tasks":[]`, `"members":[]`, `"counts":{}`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("falta %s en %s", want, raw)
		}
	}
}

func TestToolRejectsInvalidProjectID(t *testing.T) {
	result := callTool(t, &fakeReader{}, "abc")

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "project_id inválido") {
		t.Errorf("resultado = %q", text)
	}
}

func TestToolProjectNotFound(t *testing.T) {
	result := callTool(t, &fakeReader{projectErr: kernel.NotFound("proyecto no encontrado")}, uuid.NewString())

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "proyecto no encontrado") {
		t.Errorf("resultado = %q", text)
	}
}
