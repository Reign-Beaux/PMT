package changestatus

import (
	"context"
	"encoding/json"
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
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "change_project_status", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRequiresProjectIDAndStatus(t *testing.T) {
	repo, _ := newRepoWith(t)
	tools, err := connect(t, repo).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	if !slices.Contains(required, any("project_id")) || !slices.Contains(required, any("status")) {
		t.Errorf("required = %v", required)
	}
}

func TestToolChangesStatus(t *testing.T) {
	repo, id := newRepoWith(t)

	result := call(t, connect(t, repo), map[string]any{"project_id": id.String(), "status": "stand_by"})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	if output.ID != id.String() || output.Status != "stand_by" || output.Name != "PMT" {
		t.Errorf("salida = %+v", output)
	}
	if repo.projects[id].Status != domain.StatusStandBy {
		t.Errorf("proyecto = %+v", repo.projects[id])
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args func(uuid.UUID) map[string]any
		want string
	}{
		{"project_id inválido", func(uuid.UUID) map[string]any {
			return map[string]any{"project_id": "abc", "status": "stand_by"}
		}, "project_id inválido"},
		{"transición no permitida", func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "status": "working"}
		}, "transición"},
		{"status desconocido", func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "status": "archived"}
		}, "desconocido"},
		{"proyecto inexistente", func(uuid.UUID) map[string]any {
			return map[string]any{"project_id": uuid.NewString(), "status": "stand_by"}
		}, "no encontrado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t)

			result := call(t, connect(t, repo), tt.args(id))

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
			if repo.updates != 0 {
				t.Error("no debió guardar el status")
			}
		})
	}
}
