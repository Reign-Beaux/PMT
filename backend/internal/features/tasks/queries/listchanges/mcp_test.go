package listchanges

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, handler *Handler) *sdk.ClientSession {
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
	return session
}

func call(t *testing.T, reader Reader, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	result, err := connect(t, New(reader)).CallTool(context.Background(), &sdk.CallToolParams{Name: "list_task_changes", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolListsChanges(t *testing.T) {
	reader := &fakeReader{histories: []domain.History{{
		Title:   "Original",
		Changes: []domain.Change{{Version: 2, Kind: domain.ChangeEdit, Title: "Otra", Reason: "Motivo"}},
	}}}

	result := call(t, reader, map[string]any{"stage_id": uuid.NewString()})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !strings.Contains(text, `"reason":"Motivo"`) || !strings.Contains(text, `"version":2`) || !strings.Contains(text, `"original":{`) {
		t.Errorf("salida = %s", text)
	}
	if reader.filter != nil {
		t.Errorf("filtro = %v", reader.filter)
	}
}

func TestToolFiltersByTask(t *testing.T) {
	reader := &fakeReader{}
	id := uuid.New()

	call(t, reader, map[string]any{"stage_id": uuid.NewString(), "task_id": id.String()})

	if reader.filter == nil || *reader.filter != id {
		t.Errorf("filtro = %v", reader.filter)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{"padre no es UUID", map[string]any{"stage_id": "abc"}, "stage_id inválido"},
		{"filtro no es UUID", map[string]any{"stage_id": uuid.NewString(), "task_id": "abc"}, "task_id inválido"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := call(t, &fakeReader{}, tt.args)

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
		})
	}
}
