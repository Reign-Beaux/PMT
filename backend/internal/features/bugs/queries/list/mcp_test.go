package list

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/bugs/domain"
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

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "list_bugs", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolWorks(t *testing.T) {

	result := call(t, New(fakeReader{bugs: []domain.Bug{{Title: "A"}}}), map[string]any{"project_id": uuid.NewString()})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
}

func TestToolRejectsInvalidID(t *testing.T) {

	result := call(t, New(fakeReader{bugs: []domain.Bug{{Title: "A"}}}), map[string]any{"project_id": "x"})

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "project_id inválido") {
		t.Errorf("resultado = %q", text)
	}
}
