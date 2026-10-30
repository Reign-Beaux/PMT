package list

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/domain"
	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, reader Reader) *sdk.ClientSession {
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
	return session
}

func call(t *testing.T, session *sdk.ClientSession) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "list_projects"})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolDescriptionMentionsProjectID(t *testing.T) {
	tools, err := connect(t, &fakeReader{}).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	description := tools.Tools[0].Description
	if !strings.Contains(description, "ID") || !strings.Contains(description, "create_stage") {
		t.Errorf("descripción = %q", description)
	}
}

func TestToolReturnsEmptyArray(t *testing.T) {
	result := call(t, connect(t, &fakeReader{}))

	text := result.Content[0].(*sdk.TextContent).Text
	if result.IsError || strings.TrimSpace(text) != `{"projects":[]}` {
		t.Errorf("resultado = %q", text)
	}
}

func TestToolListsProjects(t *testing.T) {
	id := uuid.New()
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := created.Add(time.Hour)
	reader := &fakeReader{projects: []domain.Project{
		{ID: id, Name: "PMT", Description: "Gestor", CreatedAt: created, UpdatedAt: updated},
	}}

	result := call(t, connect(t, reader))

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	want := toolProject{
		ID:          id.String(),
		Name:        "PMT",
		Description: "Gestor",
		CreatedAt:   "2026-01-02T03:04:05Z",
		UpdatedAt:   "2026-01-02T04:04:05Z",
	}
	if len(output.Projects) != 1 || output.Projects[0] != want {
		t.Errorf("proyectos = %+v", output.Projects)
	}
}
