package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	indexBody  = "<!doctype html><title>PMT</title>"
	secretBody = "contenido fuera de dist"
)

// distDePrueba crea un dist con index.html, un asset, un archivo externo
// hermano y un enlace simbólico que apunta fuera del dist.
func distDePrueba(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	dist := filepath.Join(base, "dist")
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(indexBody), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "assets", "app.js"), []byte("console.log(1)"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "secret.txt"), []byte(secretBody), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "secret.txt"), filepath.Join(dist, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	return dist
}

func sirve(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestHandlerSirveArchivosExistentes(t *testing.T) {
	h := Handler(distDePrueba(t))

	// /index.html recibe la redirección canónica del FileServer a "./" y la
	// raíz sirve su contenido con el tipo correcto.
	rec := sirve(t, h, http.MethodGet, "/index.html")
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "./" {
		t.Errorf("/index.html: status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}

	rec = sirve(t, h, http.MethodGet, "/")
	if rec.Code != http.StatusOK || rec.Body.String() != indexBody {
		t.Errorf("/: status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("/: Content-Type = %q", ct)
	}

	rec = sirve(t, h, http.MethodGet, "/assets/app.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Errorf("/assets/app.js: status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("/assets/app.js: Content-Type = %q", ct)
	}
}

func TestHandlerFallbackSPARutaProfundaConQuery(t *testing.T) {
	h := Handler(distDePrueba(t))

	rec := sirve(t, h, http.MethodGet, "/projects/p1/stages/s1?task=t1")
	if rec.Code != http.StatusOK || rec.Body.String() != indexBody {
		t.Errorf("ruta profunda: status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("ruta profunda: Content-Type = %q", ct)
	}

	rec = sirve(t, h, http.MethodGet, "/")
	if rec.Code != http.StatusOK || rec.Body.String() != indexBody {
		t.Errorf("/: status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestHandlerNoEscapaDeDist(t *testing.T) {
	h := Handler(distDePrueba(t))

	for _, target := range []string{
		"/../secret.txt",
		"/%2e%2e/secret.txt",
		"/assets/../../secret.txt",
		"/escape.txt",
	} {
		rec := sirve(t, h, http.MethodGet, target)
		if strings.Contains(rec.Body.String(), secretBody) {
			t.Errorf("%s: sirvió el archivo externo", target)
		}
		if rec.Code != http.StatusOK || rec.Body.String() != indexBody {
			t.Errorf("%s: status = %d, body = %q", target, rec.Code, rec.Body.String())
		}
	}
}

func TestHandlerNoListaDirectorios(t *testing.T) {
	h := Handler(distDePrueba(t))

	for _, target := range []string{"/assets/", "/assets"} {
		rec := sirve(t, h, http.MethodGet, target)
		if rec.Code != http.StatusOK || rec.Body.String() != indexBody {
			t.Errorf("%s: status = %d, body = %q", target, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "app.js") {
			t.Errorf("%s: listó el contenido del directorio", target)
		}
	}
}

func TestHandlerRechazaOtrosMetodos(t *testing.T) {
	h := Handler(distDePrueba(t))

	rec := sirve(t, h, http.MethodPost, "/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /: status = %d", rec.Code)
	}
}

func TestHandlerSinDistResponde404(t *testing.T) {
	h := Handler(filepath.Join(t.TempDir(), "no-existe"))

	rec := sirve(t, h, http.MethodGet, "/")
	if rec.Code != http.StatusNotFound {
		t.Errorf("dist inexistente: status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "frontend no compilado") {
		t.Errorf("dist inexistente: body = %q", rec.Body.String())
	}
}

func TestHandlerDistSinIndexResponde404(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "otro.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := Handler(dist)

	rec := sirve(t, h, http.MethodGet, "/")
	if rec.Code != http.StatusNotFound {
		t.Errorf("dist sin index.html: status = %d", rec.Code)
	}
}
