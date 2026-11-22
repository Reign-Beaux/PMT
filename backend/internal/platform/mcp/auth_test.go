package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

var testActor = kernel.Actor{Name: "agente", Profile: kernel.ProfileTL}

type fakeAuthenticator struct{}

// AuthenticateToken aplica el perfil declarado como lo hace app: el token de
// usuario toma cualquiera y el de agente solo el suyo.
func (fakeAuthenticator) AuthenticateToken(_ context.Context, token string, declared kernel.Profile) (kernel.Actor, error) {
	switch token {
	case "bueno":
		return kernel.AsAgent(testActor, declared)
	case "pmt_ana":
		return kernel.AsUser("ana", declared), nil
	}
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

// AuthenticateSession nunca se usa en el MCP: solo acepta tokens.
func (fakeAuthenticator) AuthenticateSession(context.Context, string) (kernel.Actor, error) {
	return testActor, nil
}

type whoamiInput struct{}

type whoamiOutput struct {
	Name    string `json:"name"`
	Profile string `json:"profile"`
}

// serverWithWhoami publica por HTTP sin estado una herramienta que devuelve su actor.
func serverWithWhoami() http.Handler {
	server := NewServer()
	sdk.AddTool(server, &sdk.Tool{Name: "whoami"}, func(_ context.Context, req *sdk.CallToolRequest, _ whoamiInput) (*sdk.CallToolResult, whoamiOutput, error) {
		actor := ActorFrom(req)
		return nil, whoamiOutput{Name: actor.Name, Profile: string(actor.Profile)}, nil
	})
	return Authenticate(fakeAuthenticator{})(Handler(server))
}

func callWhoami(handler http.Handler, authorization string) *httptest.ResponseRecorder {
	return callWhoamiAt(handler, "/mcp", authorization)
}

func callWhoamiAt(handler http.Handler, target, authorization string) *httptest.ResponseRecorder {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-06-18")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name   string
		header string
		status int
		actor  string
	}{
		{"sin cabecera", "", http.StatusUnauthorized, ""},
		{"variable vacía", "Bearer", http.StatusUnauthorized, ""},
		{"válido", "Bearer bueno", http.StatusOK, testActor.Name},
		{"inválido", "Bearer malo", http.StatusUnauthorized, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := callWhoami(serverWithWhoami(), tt.header)

			if rec.Code != tt.status {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body)
			}
			if tt.actor != "" && !strings.Contains(rec.Body.String(), `\"name\":\"`+tt.actor) && !strings.Contains(rec.Body.String(), `"name":"`+tt.actor) {
				t.Errorf("cuerpo = %s", rec.Body)
			}
		})
	}
}

func TestAuthenticateDeclaredProfile(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		header  string
		status  int
		actor   string
		message string
	}{
		{"usuario sin perfil", "/mcp", "Bearer pmt_ana", http.StatusOK, `"name":"ana","profile":"User"`, ""},
		{"usuario como TL", "/mcp?profile=tl", "Bearer pmt_ana", http.StatusOK, `"name":"ana","profile":"TL"`, ""},
		{"sin distinguir mayúsculas", "/mcp?profile=QAP", "Bearer pmt_ana", http.StatusOK, `"name":"ana","profile":"QAP"`, ""},
		{"user equivale a no declarar", "/mcp?profile=user", "Bearer pmt_ana", http.StatusOK, `"name":"ana","profile":"User"`, ""},
		{"perfil desconocido", "/mcp?profile=xx", "Bearer pmt_ana", http.StatusBadRequest, "", "perfil desconocido: usa pp, pc, tl"},
		{"agente con su perfil", "/mcp?profile=TL", "Bearer bueno", http.StatusOK, `"name":"agente","profile":"TL"`, ""},
		{"agente con otro perfil", "/mcp?profile=pp", "Bearer bueno", http.StatusBadRequest, "", "declara ese perfil o ninguno"},
		{"sin token con perfil", "/mcp?profile=xx", "", http.StatusUnauthorized, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := callWhoamiAt(serverWithWhoami(), tt.target, tt.header)

			if rec.Code != tt.status {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body)
			}
			body := strings.ReplaceAll(rec.Body.String(), `\"`, `"`)
			if !strings.Contains(body, tt.actor) || !strings.Contains(body, tt.message) {
				t.Errorf("cuerpo = %s", rec.Body)
			}
			if strings.Contains(body, "oauth") {
				t.Errorf("el mensaje no debe decir oauth: %s", rec.Body)
			}
		})
	}
}

func TestVerifierKeepsTechnicalErrors(t *testing.T) {
	want := errors.New("base de datos caída")
	check := verifier(authenticatorFunc(func(context.Context, string) (kernel.Actor, error) {
		return kernel.Actor{}, want
	}))

	_, err := check(context.Background(), "x", nil)

	if !errors.Is(err, want) || errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("error = %v", err)
	}
}

func TestVerifierMarksTokenChannel(t *testing.T) {
	check := verifier(authenticatorFunc(func(context.Context, string) (kernel.Actor, error) {
		return kernel.Actor{Name: "ana", Profile: kernel.ProfileUser}, nil
	}))

	info, err := check(context.Background(), "pmt_x", nil)

	if err != nil {
		t.Fatal(err)
	}
	if actor, _ := info.Extra[actorKey].(kernel.Actor); actor.Channel != kernel.ChannelToken || actor.InBrowser() {
		t.Errorf("actor = %+v", actor)
	}
}

type authenticatorFunc func(context.Context, string) (kernel.Actor, error)

func (f authenticatorFunc) AuthenticateToken(ctx context.Context, token string, _ kernel.Profile) (kernel.Actor, error) {
	return f(ctx, token)
}

func (f authenticatorFunc) AuthenticateSession(ctx context.Context, session string) (kernel.Actor, error) {
	return f(ctx, session)
}

func TestActorFromWithoutExtra(t *testing.T) {
	if got := ActorFrom(&sdk.CallToolRequest{}); got != (kernel.Actor{}) {
		t.Errorf("actor = %+v", got)
	}
}
