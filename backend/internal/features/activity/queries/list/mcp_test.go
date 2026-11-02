package list

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/activity/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func call(t *testing.T, reader Reader, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(reader).RegisterTool(server)

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "list_comments", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolLists(t *testing.T) {
	id := uuid.NewString()
	reader := &fakeReader{comments: []domain.Comment{{Body: "Decisión"}}}

	result := call(t, reader, map[string]any{"item_kind": "project", "item_id": id})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, "Decisión") {
		t.Errorf("salida = %s", text)
	}
}

func TestToolEmptyIsArray(t *testing.T) {
	result := call(t, &fakeReader{}, map[string]any{"item_kind": "task", "item_id": uuid.NewString()})

	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, `"comments":[]`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidID(t *testing.T) {
	if result := call(t, &fakeReader{}, map[string]any{"item_kind": "task", "item_id": "abc"}); !result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
}
