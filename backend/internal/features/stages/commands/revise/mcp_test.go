package revise

import (
	"context"
	"strings"
	"testing"

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
	result, err := connect(t, New(repo)).CallTool(context.Background(), &sdk.CallToolParams{
		Name: "revise_stage",
		Arguments: map[string]any{
			"stage_id": id, "name": "Otra", "description": "Detalle", "reason": "Motivo", "requested_by": "PM",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRevises(t *testing.T) {
	repo := newRepo()

	result := call(t, repo, repo.current.ID.String())

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	if text := result.Content[0].(*sdk.TextContent).Text; !strings.Contains(text, `"version":2`) {
		t.Errorf("salida = %s", text)
	}
}

func TestToolRejectsInvalidID(t *testing.T) {
	result := call(t, newRepo(), "abc")

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "stage_id inválido") {
		t.Errorf("resultado = %q", text)
	}
}
