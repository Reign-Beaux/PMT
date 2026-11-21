package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
	"pmt/internal/platform/web"
)

type rejectAll struct{}

type pingOK struct{}

func (pingOK) Ping(context.Context) error { return nil }

type pingFalla struct{}

func (pingFalla) Ping(context.Context) error { return errors.New("sin base") }

func (rejectAll) Authenticate(context.Context, string) (kernel.Actor, error) {
	return kernel.Actor{}, kernel.ErrUnauthenticated
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

	for _, path := range []string{"/api", "/api/projects", "/api/ruta-inexistente", "/mcp"} {
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

func TestRoutesSinTokenNoCaenAlFrontend(t *testing.T) {
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, pingOK{}, frontendDePrueba())

	// Sin token la petición pasa como usuario y responde la API/MCP (el mux de
	// prueba está vacío: 404 o 405), nunca index.html.
	for _, path := range []string{"/api", "/api/ruta-inexistente", "/mcp"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if strings.Contains(rec.Body.String(), "index de prueba") {
			t.Errorf("%s: status = %d, el frontend tapó la ruta", path, rec.Code)
		}
		if rec.Code == http.StatusOK {
			t.Errorf("%s: status = %d, se esperaba una respuesta de la API/MCP", path, rec.Code)
		}
	}

	for _, path := range []string{"/", "/projects/p1/stages/s1"} {
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
	handler := routes(api, mcp.NewServer(), rejectAll{}, pingOK{}, web.Handler(t.TempDir()))

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
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s: status = %d, cuerpo = %q", tc.path, rec.Code, rec.Body.String())
		}
	}
}
