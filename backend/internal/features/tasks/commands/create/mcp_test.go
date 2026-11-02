package create

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, repo Repository) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(repo, fakeStages{}).RegisterTool(server)

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
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "create_task", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRequiresStageTitleAndDescription(t *testing.T) {
	tools, err := connect(t, &fakeRepo{}).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	for _, field := range []string{"stage_id", "title", "description"} {
		if !slices.Contains(required, any(field)) {
			t.Errorf("required = %v", required)
		}
	}
}

func TestToolCreatesTask(t *testing.T) {
	repo := &fakeRepo{}
	stageID := uuid.New()

	result := call(t, connect(t, repo), map[string]any{
		"stage_id": stageID.String(), "title": "Revisar", "description": "Detalle",
	})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if len(repo.saved) != 1 || repo.saved[0].Title != "Revisar" || repo.saved[0].StageID != stageID {
		t.Errorf("guardadas = %+v", repo.saved)
	}
}

func TestToolRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		stage string
		title string
		err   error
		want  string
	}{
		{"etapa no es UUID", "abc", "Revisar", nil, "stage_id inválido"},
		{"título vacío", uuid.NewString(), " ", nil, "obligatorio"},
		{"etapa inexistente", uuid.NewString(), "Revisar", domain.ErrStageNotFound, "etapa no encontrada"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{err: tt.err}

			result := call(t, connect(t, repo), map[string]any{
				"stage_id": tt.stage, "title": tt.title, "description": "Detalle",
			})

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
			if len(repo.saved) != 0 {
				t.Error("no debió guardar la tarea")
			}
		})
	}
}
