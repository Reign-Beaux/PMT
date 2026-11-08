package changeproductversion

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
	platformmcp "pmt/internal/platform/mcp"
)

func connect(t *testing.T, repo Repository, actor kernel.Actor) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := platformmcp.NewServer()
	platformmcp.ActAs(server, actor)
	New(repo).RegisterTool(server)

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

func call(t *testing.T, session *sdk.ClientSession, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "change_product_version", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestToolRequiresAllFields(t *testing.T) {
	repo, _ := newRepoWith(t, domain.StatusWorking, "1.0.0")
	tools, err := connect(t, repo, pp).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	schema, ok := tools.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("esquema = %T", tools.Tools[0].InputSchema)
	}
	required, _ := schema["required"].([]any)
	for _, field := range []string{"project_id", "product_version", "reason"} {
		if !slices.Contains(required, any(field)) {
			t.Errorf("required = %v, falta %s", required, field)
		}
	}
}

func TestToolChangesVersion(t *testing.T) {
	repo, id := newRepoWith(t, domain.StatusWorking, "1.0.0")

	result := call(t, connect(t, repo, pp), map[string]any{"project_id": id.String(), "product_version": "1.0.1", "reason": "bug corregido"})

	if result.IsError {
		t.Fatalf("resultado = %+v", result)
	}
	var output toolOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	if output.ID != id.String() || output.ProductVersion != "1.0.1" || output.Status != "working" {
		t.Errorf("salida = %+v", output)
	}
	if len(repo.recorded) != 1 || repo.recorded[0].Reason != "bug corregido" {
		t.Errorf("registros = %+v", repo.recorded)
	}
}

func TestToolRejects(t *testing.T) {
	tests := []struct {
		name   string
		actor  kernel.Actor
		status domain.Status
		args   func(uuid.UUID) map[string]any
		want   string
	}{
		{"PC", kernel.Actor{Name: "pc", Profile: kernel.ProfilePC}, domain.StatusWorking, func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "product_version": "1.1.0", "reason": "x"}
		}, "forbidden:"},
		{"TL", kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}, domain.StatusWorking, func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "product_version": "1.1.0", "reason": "x"}
		}, "forbidden:"},
		{"project_id inválido", pp, domain.StatusWorking, func(uuid.UUID) map[string]any {
			return map[string]any{"project_id": "abc", "product_version": "1.1.0", "reason": "x"}
		}, "project_id inválido"},
		{"fuera de working", pp, domain.StatusPlanning, func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "product_version": "1.1.0", "reason": "x"}
		}, "no está en working"},
		{"formato", pp, domain.StatusWorking, func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "product_version": "1.01.0", "reason": "x"}
		}, "X.Y.Z"},
		{"no mayor", pp, domain.StatusWorking, func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "product_version": "1.0.0", "reason": "x"}
		}, "mayor"},
		{"motivo vacío", pp, domain.StatusWorking, func(id uuid.UUID) map[string]any {
			return map[string]any{"project_id": id.String(), "product_version": "1.1.0", "reason": ""}
		}, "motivo"},
		{"proyecto inexistente", pp, domain.StatusWorking, func(uuid.UUID) map[string]any {
			return map[string]any{"project_id": uuid.NewString(), "product_version": "1.1.0", "reason": "x"}
		}, "no encontrado"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, id := newRepoWith(t, tt.status, "1.0.0")

			result := call(t, connect(t, repo, tt.actor), tt.args(id))

			text := result.Content[0].(*sdk.TextContent).Text
			if !result.IsError || !strings.Contains(text, tt.want) {
				t.Errorf("resultado = %q", text)
			}
			if len(repo.recorded) != 0 || repo.projects[id].ProductVersion != "1.0.0" {
				t.Error("no debió guardar la versión")
			}
		})
	}
}
