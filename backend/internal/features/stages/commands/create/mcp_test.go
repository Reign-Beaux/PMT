package create

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/domain"
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
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "create_stage", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRequiresProjectNameAndDescription(t *testing.T) {
	tools, err := connect(t, &fakeRepo{}).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	for _, field := range []string{"project_id", "name", "description"} {
		if !slices.Contains(required, any(field)) {
			t.Errorf("required = %v", required)
		}
	}
}

func TestToolCreatesStage(t *testing.T) {
	repo := &fakeRepo{}
	projectID := uuid.New()

	result := call(t, connect(t, repo), map[string]any{
		"project_id": projectID.String(), "name": "Diseño", "description": "Primera etapa",
	})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if len(repo.saved) != 1 || repo.saved[0].Name != "Diseño" || repo.saved[0].ProjectID != projectID {
		t.Errorf("guardadas = %+v", repo.saved)
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, `"status":"planning"`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		project string
		stage   string
		err     error
		want    string
	}{
		{"proyecto no es UUID", "abc", "Diseño", nil, "project_id inválido"},
		{"nombre vacío", uuid.NewString(), " ", nil, "obligatorio"},
		{"proyecto inexistente", uuid.NewString(), "Diseño", domain.ErrProjectNotFound, "proyecto no encontrado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{err: tt.err}

			result := call(t, connect(t, repo), map[string]any{
				"project_id": tt.project, "name": tt.stage, "description": "Etapa",
			})

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar la etapa")
			}
		})
	}
}
