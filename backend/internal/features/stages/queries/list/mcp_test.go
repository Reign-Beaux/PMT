package list

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func callTool(t *testing.T, reader Reader, projectID string) *sdk.CallToolResult {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	New(reader).RegisterTool(server)

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

	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "list_stages",
		Arguments: map[string]any{"project_id": projectID},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolListsStagesInOrder(t *testing.T) {
	reader := &fakeReader{stages: []domain.Stage{
		{ID: uuid.New(), Name: "A", Position: 1},
		{ID: uuid.New(), Name: "B", Position: 2},
	}}

	result := callTool(t, reader, uuid.NewString())

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output toolOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Stages) != 2 || output.Stages[1].Name != "B" || output.Stages[1].Position != 2 {
		t.Errorf("salida = %+v", output)
	}
}

func TestToolReturnsEmptyArray(t *testing.T) {
	result := callTool(t, &fakeReader{}, uuid.NewString())

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"stages":[]`) {
		t.Errorf("salida = %s", raw)
	}
}

func TestToolRejectsInvalidProjectID(t *testing.T) {
	result := callTool(t, &fakeReader{}, "abc")

	text := result.Content[0].(*sdk.TextContent).Text
	if !result.IsError || !strings.Contains(text, "project_id inválido") {
		t.Errorf("resultado = %q", text)
	}
}
