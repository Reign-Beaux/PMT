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

func (fakeAuthenticator) Authenticate(_ context.Context, token string) (kernel.Actor, error) {
	if token == "bueno" {
		return testActor, nil
	}
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

type whoamiInput struct{}

type whoamiOutput struct {
	Name string `json:"name"`
}

// serverWithWhoami publica por HTTP sin estado una herramienta que devuelve su actor.
func serverWithWhoami() http.Handler {
	server := NewServer()
	sdk.AddTool(server, &sdk.Tool{Name: "whoami"}, func(_ context.Context, req *sdk.CallToolRequest, _ whoamiInput) (*sdk.CallToolResult, whoamiOutput, error) {
		return nil, whoamiOutput{Name: ActorFrom(req).Name}, nil
	})
	return Authenticate(fakeAuthenticator{})(Handler(server))
}

func callWhoami(handler http.Handler, authorization string) *httptest.ResponseRecorder {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
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
		{"sin cabecera", "", http.StatusOK, kernel.Owner.Name},
		{"variable vacía", "Bearer", http.StatusOK, kernel.Owner.Name},
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

type authenticatorFunc func(context.Context, string) (kernel.Actor, error)

func (f authenticatorFunc) Authenticate(ctx context.Context, token string) (kernel.Actor, error) {
	return f(ctx, token)
}

func TestActorFromWithoutExtra(t *testing.T) {
	if got := ActorFrom(&sdk.CallToolRequest{}); got != kernel.Owner {
		t.Errorf("actor = %+v", got)
	}
}
