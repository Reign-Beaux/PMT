package retire

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, handler *Handler) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	handler.RegisterTool(server)

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

func call(t *testing.T, repo Repository, id string) *sdk.CallToolResult {
	t.Helper()
	result, err := connect(t, newHandler(repo)).CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "retire_stage",
		Arguments: map[string]any{"stage_id": id, "reason": "Motivo", "requested_by": "PM"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRetires(t *testing.T) {
	id := uuid.New()

	result := call(t, &fakeRepo{existing: id}, id.String())

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, id.String()) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidID(t *testing.T) {
	result := call(t, &fakeRepo{}, "abc")

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "stage_id inválido") {
		t.Errorf("resultado = %q", text)
	}
}
