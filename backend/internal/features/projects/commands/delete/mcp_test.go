package delete

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, repo Repository) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
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
	return session
}

func call(t *testing.T, session *sdk.ClientSession, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "delete_project", Arguments: args})
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

func TestToolRequiresID(t *testing.T) {
	requireFields(t, connect(t, &fakeRepo{}), "project_id")
}

func TestToolDeletes(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{existing: id, status: domain.StatusPlanning}

	result := call(t, connect(t, repo), map[string]any{"project_id": id.String()})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if !repo.deleted {
		t.Error("debió eliminar el proyecto")
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, `"deleted_id":"`+id.String()+`"`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name   string
		id     string
		status domain.Status
		err    error
		want   string
	}{
		{"id no es UUID", "abc", domain.StatusPlanning, nil, "project_id inválido"},
		{"no existe", uuid.NewString(), domain.StatusPlanning, nil, "proyecto no encontrado"},
		{"fuera de planning", id.String(), domain.StatusStandBy, nil, "el proyecto no está en planning"},
		{"con etapas", id.String(), domain.StatusPlanning, domain.ErrProjectHasStages, "el proyecto tiene etapas"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{existing: id, status: tt.status, err: tt.err}

			result := call(t, connect(t, repo), map[string]any{"project_id": tt.id})

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
			if repo.deleted {
				t.Error("no debió eliminar el proyecto")
			}
		})
	}
}
