package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
)

type rejectAll struct{}

func (rejectAll) Authenticate(context.Context, string) (kernel.Actor, error) {
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

func TestRoutesRejectInvalidToken(t *testing.T) {
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, frontendDePrueba())

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
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, frontendDePrueba())

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
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{}, frontendDePrueba())

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
