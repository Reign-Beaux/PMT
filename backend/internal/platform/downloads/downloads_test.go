package downloads

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func siguiente() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("frontend")) })
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestConBinarios(t *testing.T) {
	dist := t.TempDir()
	for name, body := range map[string]string{"pmt-linux-amd64": "binario", Sums: "abc  pmt-linux-amd64\n", "otro": "secreto"} {
		if err := os.WriteFile(filepath.Join(dist, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := Mount(dist, "https://pmt.example.com/", siguiente())

	rec := get(h, "/install.sh")
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, `SERVIDOR="https://pmt.example.com"`) || strings.Contains(body, "__PMT_SERVER__") || !strings.HasPrefix(body, "#!/bin/sh") {
		t.Fatalf("install.sh: %d %s", rec.Code, body)
	}
	if rec := get(h, "/descargas/pmt-linux-amd64"); rec.Code != 200 || rec.Body.String() != "binario" || rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("binario: %d %q %s", rec.Code, rec.Body, rec.Header())
	}
	if rec := get(h, "/descargas/SHA256SUMS"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "pmt-linux-amd64") {
		t.Fatalf("sumas: %d", rec.Code)
	}
	for _, path := range []string{"/descargas/otro", "/descargas/pmt-darwin-arm64", "/descargas/..%2fotro", "/descargas/"} {
		if rec := get(h, path); rec.Code == 200 && rec.Body.String() != "frontend" {
			t.Errorf("%s: %d %q", path, rec.Code, rec.Body)
		}
	}
	if rec := get(h, "/projects"); rec.Body.String() != "frontend" {
		t.Fatalf("el resto va al frontend: %q", rec.Body)
	}
}

func TestSinBinariosElServidorSigue(t *testing.T) {
	for _, dist := range []string{"", filepath.Join(t.TempDir(), "no-existe"), t.TempDir()} {
		h := Mount(dist, "https://pmt.example.com", siguiente())
		rec := get(h, "/install.sh")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "no tiene los binarios del CLI") || !strings.Contains(rec.Body.String(), "exit 1") {
			t.Fatalf("%q install.sh: %d %s", dist, rec.Code, rec.Body)
		}
		if rec := get(h, "/descargas/pmt-linux-amd64"); rec.Code != http.StatusNotFound {
			t.Fatalf("%q binario: %d", dist, rec.Code)
		}
		if rec := get(h, "/"); rec.Body.String() != "frontend" {
			t.Fatalf("%q frontend: %q", dist, rec.Body)
		}
	}
}

func TestBinariosCompiladosDespuesDeArrancar(t *testing.T) {
	dist := t.TempDir()
	h := Mount(dist, "https://pmt.example.com", siguiente())
	if !strings.Contains(get(h, "/install.sh").Body.String(), "no tiene los binarios") {
		t.Fatal("sin binarios")
	}
	if err := os.WriteFile(filepath.Join(dist, Sums), []byte("x  pmt-linux-amd64\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(get(h, "/install.sh").Body.String(), "SERVIDOR=") {
		t.Fatal("con binarios, sin reiniciar")
	}
}
