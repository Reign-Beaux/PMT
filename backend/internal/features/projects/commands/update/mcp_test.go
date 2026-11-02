package update

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
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "update_project", Arguments: args})
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

	requireFields(t, connect(t, repo), "project_id", "name", "description", "product_version")
}

func TestToolUpdatesProject(t *testing.T) {
	repo, id := newRepoWith(t)

	result := call(t, connect(t, repo), map[string]any{
		"project_id": id.String(), "name": "PMT v4", "description": "Otro", "product_version": "1.0.0",
	})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if repo.projects[id].Name != "PMT v4" || repo.projects[id].ProductVersion != "1.0.0" {
		t.Errorf("proyecto = %+v", repo.projects[id])
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, `"product_version":"1.0.0"`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	repo, id := newRepoWith(t)
	p := repo.projects[id]
	p.Status = domain.StatusStandBy
	p.ProductVersion = "1.0.0"
	stopped := uuid.New()
	p.ID = stopped
	repo.projects[stopped] = p

	tests := []struct {
		name    string
		project string
		value   string
		want    string
	}{
		{"id no es UUID", "abc", "PMT", "project_id inválido"},
		{"nombre vacío", id.String(), " ", "obligatorio"},
		{"no existe", uuid.NewString(), "PMT", "proyecto no encontrado"},
		{"versión fuera de planning", stopped.String(), "Otro", "la versión de producto no se edita"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := call(t, connect(t, repo), map[string]any{
				"project_id": tt.project, "name": tt.value, "description": "Gestor", "product_version": "",
			})

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
		})
	}
}
