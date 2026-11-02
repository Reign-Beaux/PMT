package add

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

// call llama a la herramienta como TL, con una tarea que existe si itemID es válido.
func call(t *testing.T, itemID, kind string) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	id, _ := uuid.Parse(itemID)
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, tl)
	newHandler(&fakeRepo{}, id).RegisterTool(server)

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
		Name:      "add_comment",
		Arguments: map[string]any{"item_kind": kind, "item_id": itemID, "body": "Va a QA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolAdds(t *testing.T) {
	id := uuid.NewString()

	result := call(t, id, "task")

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !strings.Contains(text, id) || !strings.Contains(text, `"actor_name":"tl"`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejects(t *testing.T) {
	if result := call(t, "abc", "task"); !result.IsError {
		t.Fatalf("id inválido: %+v", result)
	}
	if result := call(t, uuid.NewString(), "finding"); !result.IsError {
		t.Fatalf("tipo desconocido: %+v", result)
	}
}
