package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/platform/mcp"
)

const contractsGlob = "../../api/features/*/*.mcp.json"

// publishedTools devuelve lo que el servidor publica, indexado por herramienta.
// El pool nulo basta porque listar herramientas no llega a la base de datos.
func publishedTools(t *testing.T) map[string]any {
	t.Helper()
	ctx := context.Background()
	server := mcp.NewServer()
	registerFeatures(http.NewServeMux(), server, nil)

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

	tools := map[string]any{}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(tool)
		if err != nil {
			t.Fatal(err)
		}
		var published any
		if err := json.Unmarshal(raw, &published); err != nil {
			t.Fatal(err)
		}
		tools[tool.Name] = published
	}
	return tools
}

// documentedTools devuelve lo que documentan los contratos, indexado por herramienta.
func documentedTools(t *testing.T) map[string]any {
	t.Helper()
	paths, err := filepath.Glob(contractsGlob)
	if err != nil {
		t.Fatal(err)
	}

	tools := map[string]any{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var contract []map[string]any
		if err := json.Unmarshal(raw, &contract); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, tool := range contract {
			name, _ := tool["name"].(string)
			if _, repeated := tools[name]; repeated {
				t.Errorf("%s: herramienta %q documentada más de una vez", path, name)
			}
			tools[name] = tool
		}
	}
	return tools
}

func TestMCPContractsMatchPublishedTools(t *testing.T) {
	published := publishedTools(t)
	documented := documentedTools(t)

	for name, got := range published {
		want, ok := documented[name]
		if !ok {
			t.Errorf("herramienta %q registrada pero no documentada", name)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("herramienta %q desalineada:\npublicada:  %s\ndocumentada: %s", name, toJSON(t, got), toJSON(t, want))
		}
	}
	for name := range documented {
		if _, ok := published[name]; !ok {
			t.Errorf("herramienta %q documentada pero no registrada", name)
		}
	}
}

func toJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
