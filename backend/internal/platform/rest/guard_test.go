package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/kernel"
)

func TestGuardRevisaConValoresDeRutaYCuerpo(t *testing.T) {
	mux := http.NewServeMux()
	var handlerBody string
	mux.HandleFunc("POST /api/stages/{id}/tasks", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		handlerBody = string(raw)
		w.WriteHeader(http.StatusCreated)
	})
	var gotPattern, gotID string
	var gotBody map[string]any
	check := func(r *http.Request, pattern string) error {
		gotPattern, gotID, gotBody = pattern, r.PathValue("id"), PeekJSON(r)
		if r.PathValue("id") == "ajena" {
			return kernel.NotFound("etapa no encontrada")
		}
		return nil
	}
	handler := Guard(mux, check)(mux)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/stages/s1/tasks", strings.NewReader(`{"title":"x"}`)))

	if rec.Code != http.StatusCreated || gotPattern != "POST /api/stages/{id}/tasks" || gotID != "s1" || gotBody["title"] != "x" {
		t.Fatalf("status = %d, patrón = %q, id = %q, cuerpo = %v", rec.Code, gotPattern, gotID, gotBody)
	}
	if handlerBody != `{"title":"x"}` {
		t.Errorf("el handler recibió %q", handlerBody)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/stages/ajena/tasks", strings.NewReader(`{}`)))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "etapa no encontrada") {
		t.Errorf("rechazo: %d %s", rec.Code, rec.Body)
	}
}

func TestGuardDejaPasarLoQueNoAtiendeNingunaRuta(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", func(http.ResponseWriter, *http.Request) {})
	revisadas := 0
	handler := Guard(mux, func(*http.Request, string) error { revisadas++; return nil })(mux)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nada", nil))

	if rec.Code != http.StatusNotFound || revisadas != 0 {
		t.Errorf("status = %d, revisadas = %d", rec.Code, revisadas)
	}
}

func TestPeekJSONIgnoraLoQueNoEsUnObjeto(t *testing.T) {
	for _, body := range []string{"", "[1]", "{malo", strings.Repeat("a", maxBodyBytes+2)} {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		if got := PeekJSON(r); got != nil {
			t.Errorf("%.10q: %v", body, got)
		}
		if raw, _ := io.ReadAll(r.Body); string(raw) != body {
			t.Errorf("%.10q: el cuerpo cambió", body)
		}
	}
}
