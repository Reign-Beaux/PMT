package list

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func callTool(t *testing.T, handler *Handler, stageID string) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	handler.RegisterTool(server)

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
		Name:      "list_tasks",
		Arguments: map[string]any{"stage_id": stageID},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolListsTasks(t *testing.T) {
	reader := &fakeReader{tasks: []domain.Task{{ID: uuid.New(), Title: "Revisar", Status: domain.StatusBacklog, Version: 2}}}

	result := callTool(t, New(reader, fakeStages{}), uuid.NewString())

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
	if len(output.Tasks) != 1 || output.Tasks[0].Title != "Revisar" || output.Tasks[0].Status != "backlog" || output.Tasks[0].Version != 2 {
		t.Errorf("salida = %+v", output)
	}
}

func TestToolReturnsEmptyArray(t *testing.T) {
	result := callTool(t, New(&fakeReader{}, fakeStages{}), uuid.NewString())

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"tasks":[]`) {
		t.Errorf("salida = %s", raw)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		stage  string
		stages fakeStages
		want   string
	}{
		{"etapa no es UUID", "abc", fakeStages{}, "stage_id inválido"},
		{"etapa inexistente", uuid.NewString(), fakeStages{err: domain.ErrStageNotFound}, "etapa no encontrada"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := callTool(t, New(&fakeReader{}, tt.stages), tt.stage)

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
		})
	}
}
