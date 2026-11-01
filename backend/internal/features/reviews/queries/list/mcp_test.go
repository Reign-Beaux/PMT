package list

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/reviews/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func call(t *testing.T, reader Reader, tasks TaskLister, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(reader, tasks).RegisterTool(server)

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "list_findings", Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolListsByStageWithItem(t *testing.T) {
	taskID := uuid.New()
	reader := &fakeReader{findings: []domain.Finding{{ID: uuid.New(), ItemKind: domain.ItemTask, ItemID: taskID}}}

	result := call(t, reader, fakeTasks{ids: []uuid.UUID{taskID}}, map[string]any{"stage_id": uuid.NewString()})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !strings.Contains(text, `"item_kind":"task"`) || !strings.Contains(text, taskID.String()) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name      string
		arguments map[string]any
	}{
		{"sin modo", map[string]any{}},
		{"los dos modos", map[string]any{"item_kind": "task", "item_id": id, "stage_id": id}},
		{"etapa inválida", map[string]any{"stage_id": "abc"}},
		{"elemento inválido", map[string]any{"item_kind": "task", "item_id": "abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := call(t, &fakeReader{}, fakeTasks{}, tt.arguments); !result.IsError {
				t.Fatalf("resultado = %+v", result)
			}
		})
	}
}
