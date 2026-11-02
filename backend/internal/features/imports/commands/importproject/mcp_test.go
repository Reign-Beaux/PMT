package importproject

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

func call(t *testing.T, repo Repository, path string) *sdk.CallToolResult {
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

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "import_project", Arguments: map[string]any{"path": path}})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolImportsProject(t *testing.T) {
	repo := &fakeRepo{}

	result := call(t, repo, writeFile(t, validFile))

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	output, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("salida = %T", result.StructuredContent)
	}
	if output["project_name"] != "Tienda" || output["stage_count"] != float64(2) || output["task_count"] != float64(2) {
		t.Errorf("salida = %+v", output)
	}
	if output["project_id"] != repo.saved[0].Project.ID.String() {
		t.Errorf("project_id = %v", output["project_id"])
	}
}

func TestToolRejectsRelativePath(t *testing.T) {
	result := call(t, &fakeRepo{}, "proyecto.yaml")

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "ruta absoluta") {
		t.Errorf("resultado = %q", text)
	}
}

func TestToolReportsErrorLocation(t *testing.T) {
	path := writeFile(t, "project:\n  name: A\n  description: B\n  stages:\n    - name: C\n")

	result := call(t, &fakeRepo{}, path)

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "etapa 1: la descripción es obligatoria") {
		t.Errorf("resultado = %q", text)
	}
}
