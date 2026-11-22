package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
)

// statusFor da un status válido para la herramienta, para que el rechazo sea por
// perfil y no por un status desconocido.
var statusFor = map[string]string{
	"change_bug_status": "rejected",
}

// sampleArguments llena cada campo del esquema con un valor válido en forma. Los ID
// no existen: la herramienta debe rechazar el perfil antes de buscarlos.
func sampleArguments(t *testing.T, tool *sdk.Tool) map[string]any {
	t.Helper()
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}

	args := map[string]any{}
	for name, prop := range schema.Properties {
		kind := fmt.Sprint(prop.Type)
		switch {
		case strings.Contains(kind, "array"):
			args[name] = []string{uuid.NewString()}
		case strings.Contains(kind, "boolean"):
			args[name] = true
		case strings.HasSuffix(name, "_id"):
			args[name] = uuid.NewString()
		case name == "status":
			args[name] = "cancelled"
			if status, ok := statusFor[tool.Name]; ok {
				args[name] = status
			}
		case name == "resolution":
			args[name] = "fixed"
		case name == "item_kind":
			args[name] = "task"
		case name == "path":
			args[name] = "/pmt-no-existe/plan.yaml"
		default:
			args[name] = "x"
		}
	}
	return args
}

// TestWriteToolsRejectPC recorre toda herramienta de escritura registrada y exige que
// rechace a PC, que solo lee, con forbidden y antes de tocar la base: el pool es nulo,
// así que una herramienta que llegue a ella entra en pánico y la prueba lo reporta.
func TestWriteToolsRejectPC(t *testing.T) {
	assertWriteToolsReject(t, &kernel.Actor{Name: "pc-prueba", Profile: kernel.ProfilePC}, "PC")
}

// TestWriteToolsRejectUserAsPC exige lo mismo a un usuario cuya sesión declara
// PC (/mcp?profile=pc): manda el perfil declarado, no que sea usuario.
func TestWriteToolsRejectUserAsPC(t *testing.T) {
	actor := kernel.AsUser("saul", kernel.ProfilePC)
	assertWriteToolsReject(t, &actor, "un usuario como PC")
}

// TestWriteToolsRejectEmptyActor exige lo mismo para una llamada sin actor. La
// autenticación siempre deja uno; si faltara, el actor vacío no puede escribir.
func TestWriteToolsRejectEmptyActor(t *testing.T) {
	assertWriteToolsReject(t, nil, "el actor vacío")
}

func assertWriteToolsReject(t *testing.T, actor *kernel.Actor, quien string) {
	t.Helper()
	ctx := context.Background()
	server := mcp.NewServer()
	registerFeatures(http.NewServeMux(), server, nil)
	if actor != nil {
		mcp.ActAs(server, *actor)
	}
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (result sdk.Result, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("llegó a la base antes de revisar el perfil: %v", r)
				}
			}()
			return next(ctx, method, req)
		}
	})

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

	checked := 0
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		if readOnly(tool.Name) {
			continue
		}
		checked++
		t.Run(tool.Name, func(t *testing.T) {
			result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tool.Name, Arguments: sampleArguments(t, tool)})
			if err != nil {
				t.Fatalf("%s no fue rechazado: %v", quien, err)
			}
			text := ""
			if len(result.Content) > 0 {
				if content, ok := result.Content[0].(*sdk.TextContent); ok {
					text = content.Text
				}
			}
			if !result.IsError || !strings.HasPrefix(text, "forbidden: ") {
				t.Errorf("%s no fue rechazado por perfil: %q", quien, text)
			}
		})
	}
	if checked == 0 {
		t.Fatal("no se encontró ninguna herramienta de escritura")
	}
}
