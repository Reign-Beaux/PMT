package mcp

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

type echoInput struct {
	StageID string `json:"stage_id"`
}

func TestGuardRevisaAntesDeLaHerramienta(t *testing.T) {
	ctx := context.Background()
	server := NewServer()
	llamadas := 0
	sdk.AddTool(server, &sdk.Tool{Name: "echo"}, func(context.Context, *sdk.CallToolRequest, echoInput) (*sdk.CallToolResult, struct{}, error) {
		llamadas++
		return nil, struct{}{}, nil
	})
	var got kernel.Actor
	var gotTool string
	var gotArgs map[string]any
	Guard(server, func(_ context.Context, actor kernel.Actor, tool string, arguments json.RawMessage) error {
		got, gotTool = actor, tool
		_ = json.Unmarshal(arguments, &gotArgs)
		if gotArgs["stage_id"] == "ajena" {
			return kernel.NotFound("etapa no encontrada")
		}
		return nil
	})
	ActAs(server, testActor)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "echo", Arguments: map[string]any{"stage_id": "s1"}})
	if err != nil || result.IsError || llamadas != 1 || got != testActor || gotTool != "echo" {
		t.Fatalf("resultado = %+v, llamadas = %d, actor = %+v, herramienta = %q, err = %v", result, llamadas, got, gotTool, err)
	}

	result, err = session.CallTool(ctx, &sdk.CallToolParams{Name: "echo", Arguments: map[string]any{"stage_id": "ajena"}})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || text != "not_found: etapa no encontrada" || llamadas != 1 {
		t.Errorf("rechazo = %q, llamadas = %d", text, llamadas)
	}
}
