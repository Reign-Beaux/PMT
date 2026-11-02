package create

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

// call llama a la herramienta como CR, con una tarea que existe si itemID es válido.
func call(t *testing.T, itemID string) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	id, _ := uuid.Parse(itemID)
	items := &fakeItems{existing: id}
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, cr)
	New(&fakeRepo{}, items, items).RegisterTool(server)

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "create_finding",
		Arguments: map[string]any{"item_kind": "task", "item_id": itemID, "description": "Falla", "blocking": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolCreates(t *testing.T) {
	id := uuid.NewString()

	result := call(t, id)

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, id) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidID(t *testing.T) {
	if result := call(t, "abc"); !result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
}
