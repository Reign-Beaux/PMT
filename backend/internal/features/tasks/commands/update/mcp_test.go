package update

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, repo Repository, stages StageChecker) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(repo, stages).RegisterTool(server)

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

func call(t *testing.T, session *sdk.ClientSession, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "update_task", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func requireFields(t *testing.T, session *sdk.ClientSession, fields ...string) {
	t.Helper()
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	for _, field := range fields {
		if !slices.Contains(required, any(field)) {
			t.Errorf("required = %v", required)
		}
	}
}

func TestToolRequiresAllFields(t *testing.T) {
	repo, _ := newRepoWith(t)

	requireFields(t, connect(t, repo, fakeStages{}), "task_id", "title", "description")
}

func TestToolUpdatesTask(t *testing.T) {
	repo, id := newRepoWith(t)

	result := call(t, connect(t, repo, fakeStages{}), map[string]any{
		"task_id": id.String(), "title": "Probar", "description": "Otra",
	})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if repo.tasks[id].Title != "Probar" {
		t.Errorf("tarea = %+v", repo.tasks[id])
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, `"status":"backlog"`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	repo, id := newRepoWith(t)
	tests := []struct {
		name   string
		task   string
		value  string
		stages error
		want   string
	}{
		{"id no es UUID", "abc", "Revisar", nil, "task_id inválido"},
		{"título vacío", id.String(), " ", nil, "obligatorio"},
		{"no existe", uuid.NewString(), "Revisar", nil, "tarea no encontrada"},
		{"fuera de planning", id.String(), "Otra", errNotPlanning, "el proyecto no está en planning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := call(t, connect(t, repo, fakeStages{err: tt.stages}), map[string]any{
				"task_id": tt.task, "title": tt.value, "description": "Tarea",
			})

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
			if repo.tasks[id].Title != "Revisar" {
				t.Error("no debió guardar la tarea")
			}
		})
	}
}
