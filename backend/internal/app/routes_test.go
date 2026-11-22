package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/rest"
	"pmt/internal/platform/web"
)

type rejectAll struct{}

type pingOK struct{}

func (pingOK) Ping(context.Context) error { return nil }

type pingFalla struct{}

func (pingFalla) Ping(context.Context) error { return errors.New("sin base") }

func (rejectAll) AuthenticateSession(context.Context, string) (kernel.Actor, error) {
	return kernel.Actor{}, kernel.ErrSessionInvalid
}

func (rejectAll) AuthenticateToken(context.Context, string, kernel.Profile) (kernel.Actor, error) {
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

var (
	anaID   = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	saulID  = uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	ana     = kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: anaID}
	agente  = kernel.Actor{Name: "dbp-tl", Profile: kernel.ProfileTL}
	agenteC = kernel.Actor{Name: "dbp-tl", Profile: kernel.ProfileTL, UserID: saulID, UserName: "saul", Agent: true}
)

// duenoDeLaInstancia es saul: los tokens de agente actúan con su membresía.
type duenoDeLaInstancia struct{}

func (duenoDeLaInstancia) InstanceOwner(context.Context) (kernel.UserRef, error) {
	return kernel.UserRef{ID: saulID, Username: "saul", Active: true}, nil
}

// usuarios simula la feature auth: la sesión y el token de un usuario
// desactivado ya no autentican, porque la base solo devuelve usuarios activos, y
// el token actúa con el perfil que declare la sesión.
type usuarios struct{}

func (usuarios) AuthenticateSession(_ context.Context, session string) (kernel.Actor, error) {
	if session == "sesion-ana" {
		return ana, nil
	}
	return kernel.Actor{}, kernel.ErrSessionInvalid
}

func (usuarios) AuthenticateToken(_ context.Context, token string, declared kernel.Profile) (kernel.Actor, error) {
	if token == "pmt_ana" {
		actor := kernel.AsUser(ana.Name, declared)
		actor.UserID = anaID
		return actor, nil
	}
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

type agentes struct{}

func (agentes) Authenticate(_ context.Context, token string) (kernel.Actor, error) {
	if token == "token-de-agente" {
		return agente, nil
	}
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

type whoamiOutput struct {
	Name    string `json:"name"`
	Profile string `json:"profile"`
}

// rutasConActor arma las rutas con las credenciales reales (cookie, token de
// usuario y token de agente) y una API y un MCP que devuelven su actor.
func rutasConActor() http.Handler {
	api := http.NewServeMux()
	writeActor := func(w http.ResponseWriter, r *http.Request) {
		actor := rest.ActorFrom(r.Context())
		w.Write([]byte(actor.Name + "/" + string(actor.Profile)))
	}
	api.HandleFunc("GET /api/projects", writeActor)
	api.HandleFunc("GET /api/events", writeActor)
	api.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("login")) })
	api.HandleFunc("GET /api/invitations/{token}", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("invitación")) })
	api.HandleFunc("GET /api/invitations", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("lista")) })
	tools := mcp.NewServer()
	sdk.AddTool(tools, &sdk.Tool{Name: "whoami"}, func(_ context.Context, req *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, whoamiOutput, error) {
		actor := mcp.ActorFrom(req)
		return nil, whoamiOutput{Name: actor.Name, Profile: string(actor.Profile)}, nil
	})
	return routes(api, tools, credentials{users: usuarios{}, agents: agentes{}, owner: duenoDeLaInstancia{}}, pingOK{}, frontendDePrueba())
}

func llamarWhoami(handler http.Handler, authorization string, cookie string) *httptest.ResponseRecorder {
	return llamarHerramienta(handler, "/mcp", authorization, cookie, "whoami", `{}`)
}

func llamarHerramienta(handler http.Handler, target, authorization, cookie, tool, arguments string) *httptest.ResponseRecorder {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + tool + `","arguments":` + arguments + `}}`
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-06-18")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: rest.SessionCookie, Value: cookie})
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRoutesAPIConCadaCredencial(t *testing.T) {
	handler := rutasConActor()
	for _, tc := range []struct {
		name          string
		authorization string
		cookie        string
		code          int
		body          string
	}{
		{"sin credenciales", "", "", http.StatusUnauthorized, "autenticación requerida"},
		{"Bearer vacío", "Bearer ", "", http.StatusUnauthorized, "token inválido"},
		{"cookie de usuario", "", "sesion-ana", http.StatusOK, "ana/User"},
		{"token de usuario", "Bearer pmt_ana", "", http.StatusOK, "ana/User"},
		{"token de agente", "Bearer token-de-agente", "", http.StatusOK, "dbp-tl/TL"},
		{"cookie de usuario desactivado o vencida", "", "sesion-desactivada", http.StatusUnauthorized, "sesión no válida"},
		{"token de usuario desactivado o revocado", "Bearer pmt_desactivado", "", http.StatusUnauthorized, "token inválido"},
		{"token de agente revocado", "Bearer revocado", "", http.StatusUnauthorized, "token inválido"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: rest.SessionCookie, Value: tc.cookie})
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
				t.Errorf("status = %d, cuerpo = %q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRoutesMCPSoloConToken(t *testing.T) {
	handler := rutasConActor()
	for _, tc := range []struct {
		name          string
		authorization string
		cookie        string
		code          int
		actor         string
	}{
		{"sin credenciales", "", "", http.StatusUnauthorized, ""},
		{"Bearer vacío", "Bearer", "", http.StatusUnauthorized, ""},
		{"la cookie no basta", "", "sesion-ana", http.StatusUnauthorized, ""},
		{"token de usuario", "Bearer pmt_ana", "", http.StatusOK, `"name":"ana","profile":"User"`},
		{"token de agente", "Bearer token-de-agente", "", http.StatusOK, `"name":"dbp-tl","profile":"TL"`},
		{"token de usuario desactivado", "Bearer pmt_desactivado", "", http.StatusUnauthorized, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := llamarWhoami(handler, tc.authorization, tc.cookie)
			if rec.Code != tc.code {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body)
			}
			if tc.actor != "" && !strings.Contains(strings.ReplaceAll(rec.Body.String(), `\"`, `"`), tc.actor) {
				t.Errorf("cuerpo = %s", rec.Body)
			}
		})
	}
}

func TestRoutesPublicasDeAuth(t *testing.T) {
	handler := rutasConActor()
	for _, tc := range []struct {
		method, path string
		code         int
	}{
		{http.MethodPost, "/api/auth/login", http.StatusOK},
		{http.MethodGet, "/api/invitations/abc", http.StatusOK},
		{http.MethodGet, "/api/invitations", http.StatusUnauthorized},
		{http.MethodGet, "/api/auth/login", http.StatusUnauthorized},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.code {
			t.Errorf("%s %s: status = %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestRoutesRejectInvalidToken(t *testing.T) {
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, pingOK{}, frontendDePrueba())

	for _, path := range []string{"/api/projects", "/mcp"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Authorization", "Bearer malo")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d", path, rec.Code)
		}
	}
}

func TestRoutesConTokenInvalidoRechazanAPIYMCP(t *testing.T) {
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, pingOK{}, frontendDePrueba())

	for _, path := range []string{"/api", "/api/projects", "/api/ruta-inexistente", "/api/events", "/mcp"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer malo")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, se esperaba 401", path, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "index de prueba") {
			t.Errorf("%s: el frontend tapó la ruta", path)
		}
	}
}

func TestRoutesSinCredencialesResponden401YElFrontendQuedaAbierto(t *testing.T) {
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, pingOK{}, frontendDePrueba())

	for _, path := range []string{"/api", "/api/projects", "/api/ruta-inexistente", "/api/events", "/mcp"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, se esperaba 401", path, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "index de prueba") {
			t.Errorf("%s: el frontend tapó la ruta", path)
		}
	}

	for _, path := range []string{"/", "/login", "/projects/p1/stages/s1"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "index de prueba") {
			t.Errorf("%s: status = %d, body = %q", path, rec.Code, rec.Body.String())
		}
	}
}

func frontendDePrueba() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("index de prueba"))
	})
}

func TestRoutesSaludSinAutenticacion(t *testing.T) {
	for _, tc := range []struct {
		path   string
		pinger interface{ Ping(context.Context) error }
		code   int
	}{
		{"/healthz", pingFalla{}, http.StatusOK},
		{"/readyz", pingOK{}, http.StatusOK},
		{"/readyz", pingFalla{}, http.StatusServiceUnavailable},
	} {
		handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, tc.pinger, frontendDePrueba())
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Authorization", "Bearer malo")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != tc.code {
			t.Errorf("%s: status = %d, se esperaba %d", tc.path, rec.Code, tc.code)
		}
		if strings.Contains(rec.Body.String(), "index de prueba") {
			t.Errorf("%s: el frontend tapó la ruta", tc.path)
		}
	}
}

func TestRoutesSinCompiladoLaAPIYLaSaludSiguen(t *testing.T) {
	api := http.NewServeMux()
	api.HandleFunc("GET /api/projects", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("[]")) })
	handler := routes(api, mcp.NewServer(), credentials{users: usuarios{}, agents: agentes{}, owner: duenoDeLaInstancia{}}, pingOK{}, web.Handler(t.TempDir()))

	for _, tc := range []struct {
		path string
		code int
		body string
	}{
		{"/healthz", http.StatusOK, "ok"},
		{"/readyz", http.StatusOK, "ok"},
		{"/api/projects", http.StatusOK, "[]"},
		{"/projects/1", http.StatusNotFound, "frontend no compilado"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Authorization", "Bearer pmt_ana")
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s: status = %d, cuerpo = %q", tc.path, rec.Code, rec.Body.String())
		}
	}
}
