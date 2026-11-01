package changestatus

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, repo Repository, tasks TaskChecker) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, actorFor(domain.StatusInDev))
	New(repo, tasks, workingProject).RegisterTool(server)

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
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "change_stage_status", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRequiresStageIDAndStatus(t *testing.T) {
	repo, _ := newRepoWith(t, domain.StatusPlanning)
	tools, err := connect(t, repo, &fakeTasks{}).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	if !slices.Contains(required, any("stage_id")) || !slices.Contains(required, any("status")) {
		t.Errorf("required = %v", required)
	}
}

func TestToolChangesStatus(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusStandBy)

	result := call(t, connect(t, repo, &fakeTasks{}), map[string]any{"stage_id": id.String(), "status": "in_dev"})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	if output.ID != id.String() || output.Status != "in_dev" || output.Name != "Diseño" {
		t.Errorf("salida = %+v", output)
	}
	if repo.stages[id].Status != domain.StatusInDev {
		t.Errorf("etapa = %+v", repo.stages[id])
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		args func(uuid.UUID) map[string]any
		want string
	}{
		{"stage_id inválido", domain.StatusPlanning, func(uuid.UUID) map[string]any {
			return map[string]any{"stage_id": "abc", "status": "stand_by"}
		}, "stage_id inválido"},
		{"transición no permitida", domain.StatusPlanning, func(id uuid.UUID) map[string]any {
			return map[string]any{"stage_id": id.String(), "status": "done"}
		}, "transición"},
		{"status desconocido", domain.StatusPlanning, func(id uuid.UUID) map[string]any {
			return map[string]any{"stage_id": id.String(), "status": "archived"}
		}, "desconocido"},
		{"etapa inexistente", domain.StatusPlanning, func(uuid.UUID) map[string]any {
			return map[string]any{"stage_id": uuid.NewString(), "status": "stand_by"}
		}, "no encontrada"},
		{"tareas pendientes", domain.StatusInDev, func(id uuid.UUID) map[string]any {
			return map[string]any{"stage_id": id.String(), "status": "done"}
		}, "no se puede cerrar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.from)

			result := call(t, connect(t, repo, &fakeTasks{}), tt.args(id))

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
