package create

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

func call(t *testing.T, h *Handler, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	h.RegisterTool(server)

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

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "create_bug", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolWorks(t *testing.T) {

	result := call(t, New(&fakeRepo{}, fakeProjects{}), map[string]any{"project_id": uuid.NewString(), "title": "F", "description": "", "steps_to_reproduce": "P", "product_version": "1"})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
}

func TestToolRejectsInvalidID(t *testing.T) {

	result := call(t, New(&fakeRepo{}, fakeProjects{}), map[string]any{"project_id": "x", "title": "F", "description": "", "steps_to_reproduce": "P", "product_version": "1"})

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "project_id inválido") {
		t.Errorf("resultado = %q", text)
	}
}
