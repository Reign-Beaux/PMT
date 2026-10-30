package create

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

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
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "create_project", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRequiresNameAndDescription(t *testing.T) {
	tools, err := connect(t, &fakeRepo{}).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	if !slices.Contains(required, any("name")) || !slices.Contains(required, any("description")) {
		t.Errorf("required = %v", required)
	}
}

func TestToolCreatesProject(t *testing.T) {
	repo := &fakeRepo{}

	result := call(t, connect(t, repo), map[string]any{"name": "PMT", "description": "Gestor"})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if len(repo.saved) != 1 || repo.saved[0].Name != "PMT" {
		t.Errorf("guardados = %+v", repo.saved)
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	if output.Status != "planning" || output.ProductVersion != "" {
		t.Errorf("salida = %+v", output)
	}
}

func TestToolRejectsInvalidProject(t *testing.T) {
	repo := &fakeRepo{}

	result := call(t, connect(t, repo), map[string]any{"name": " ", "description": ""})

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "obligatorio") {
		t.Errorf("resultado = %q", text)
	}
	if len(repo.saved) != 0 {
		t.Error("no debió guardar el proyecto")
	}
}
