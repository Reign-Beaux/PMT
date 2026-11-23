package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeServer simula el servidor PMT: entrega un código y responde los sondeos
// con la secuencia dada.
type fakeServer struct {
	mu       sync.Mutex
	polls    []string // status de cada sondeo; el último se repite
	pollCode int      // status HTTP del sondeo, si no es 200
	machine  string
	revoked  []string
	meStatus int
	srv      *httptest.Server
}

const tokenNuevo = "pmt_" + "nuevo0000000000000000000000000000000000000000000000000000000000"

func newFakeServer(t *testing.T, polls ...string) *fakeServer {
	f := &fakeServer{polls: polls}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/cli/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.machine = body["machine"]
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"code": "ABCD-1234", "poll_secret": "secreto", "url": f.srv.URL + "/cli?codigo=ABCD-1234",
			"expires_at": time.Now().Add(10 * time.Minute), "interval": 2,
		})
	})
	mux.HandleFunc("POST /api/cli/login/poll", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["poll_secret"] != "secreto" {
			t.Errorf("secreto = %q", body["poll_secret"])
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.pollCode != 0 {
			w.WriteHeader(f.pollCode)
			w.Write([]byte(`{"error":"código no encontrado"}`))
			return
		}
		status := f.polls[0]
		if len(f.polls) > 1 {
			f.polls = f.polls[1:]
		}
		out := map[string]any{"status": status}
		if status == "authorized" {
			out["token"], out["token_id"], out["username"] = tokenNuevo, "id-nuevo", "saul"
		}
		json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("DELETE /api/tokens/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer pmt_") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.revoked = append(f.revoked, r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.meStatus != 0 {
			w.WriteHeader(f.meStatus)
			w.Write([]byte(`{"error":"autenticación requerida"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+tokenNuevo {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"username": "saul", "name": "Saúl"})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

type harness struct {
	app     *App
	out     *bytes.Buffer
	errOut  *bytes.Buffer
	opened  []string
	sleeps  int
	browser error
}

func newHarness(t *testing.T) *harness {
	h := &harness{out: &bytes.Buffer{}, errOut: &bytes.Buffer{}}
	h.app = &App{
		Stdout: h.out, Stderr: h.errOut, Home: t.TempDir(), HTTP: http.DefaultClient,
		Hostname:    func() (string, error) { return "laptop", nil },
		OpenBrowser: func(url string) error { h.opened = append(h.opened, url); return h.browser },
		Sleep:       func(context.Context, time.Duration) error { h.sleeps++; return nil },
		Now:         time.Now,
	}
	return h
}

func (h *harness) run(args ...string) int {
	return h.app.Run(context.Background(), args)
}

func TestLoginAutorizadoGuardaCredenciales(t *testing.T) {
	srv := newFakeServer(t, "pending", "pending", "authorized")
	h := newHarness(t)
	if code := h.run("login", "--servidor", srv.srv.URL+"/"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if srv.machine != "laptop" || h.sleeps != 3 || len(h.opened) != 1 || !strings.HasSuffix(h.opened[0], "/cli?codigo=ABCD-1234") {
		t.Fatalf("máquina=%q sondeos=%d abiertas=%v", srv.machine, h.sleeps, h.opened)
	}
	if !strings.Contains(h.out.String(), "ABCD-1234") || !strings.Contains(h.out.String(), "Sesión iniciada como saul") {
		t.Fatalf("salida: %s", h.out)
	}
	if strings.Contains(h.out.String()+h.errOut.String(), tokenNuevo) {
		t.Fatal("el token nunca se imprime")
	}
	creds, err := loadCredentials(h.app.Home)
	if err != nil || creds != (Credentials{Server: srv.srv.URL, Username: "saul", Token: tokenNuevo, TokenID: "id-nuevo"}) {
		t.Fatalf("credenciales = %+v, %v", creds, err)
	}

	// whoami usa el servidor guardado.
	h.out.Reset()
	if code := h.run("whoami"); code != 0 || h.out.String() != "saul (Saúl) en "+srv.srv.URL+"\n" {
		t.Fatalf("whoami %d: %q %s", code, h.out, h.errOut)
	}

	// Un segundo login sin --servidor usa el guardado y revoca el token anterior.
	creds.TokenID = "id-viejo"
	if err := saveCredentials(h.app.Home, creds); err != nil {
		t.Fatal(err)
	}
	srv.polls = []string{"authorized"}
	if code := h.run("login", "--sin-navegador"); code != 0 {
		t.Fatalf("segundo login %d: %s", code, h.errOut)
	}
	if len(h.opened) != 1 || !strings.Contains(h.out.String(), "puede ser en otro equipo") {
		t.Fatalf("con --sin-navegador no abre: %v", h.opened)
	}
	if len(srv.revoked) != 1 || srv.revoked[0] != "id-viejo" {
		t.Fatalf("revocados = %v", srv.revoked)
	}

	// logout revoca y borra.
	if code := h.run("logout"); code != 0 {
		t.Fatalf("logout %d: %s", code, h.errOut)
	}
	if len(srv.revoked) != 2 || srv.revoked[1] != "id-nuevo" {
		t.Fatalf("revocados = %v", srv.revoked)
	}
	if _, err := os.Stat(credentialsPath(h.app.Home)); !os.IsNotExist(err) {
		t.Fatalf("las credenciales siguen: %v", err)
	}
	h.out.Reset()
	if code := h.run("logout"); code != 0 || !strings.Contains(h.out.String(), "No había sesión") {
		t.Fatalf("logout sin sesión %d: %s", code, h.out)
	}
	if code := h.run("whoami"); code != 1 || !strings.Contains(h.errOut.String(), "pmt login") {
		t.Fatalf("whoami sin sesión %d: %s", code, h.errOut)
	}
}

func TestLoginSinNavegadorDisponibleMuestraLaURL(t *testing.T) {
	srv := newFakeServer(t, "authorized")
	h := newHarness(t)
	h.browser = errors.New("sin sesión gráfica")
	if code := h.run("login", "--servidor", srv.srv.URL); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if !strings.Contains(h.out.String(), "Abre en un navegador (puede ser en otro equipo): "+srv.srv.URL+"/cli?codigo=ABCD-1234") {
		t.Fatalf("salida: %s", h.out)
	}
}

func TestLoginRespuestasFinalesDelSondeo(t *testing.T) {
	for _, tc := range []struct {
		name     string
		polls    []string
		pollCode int
		want     string
	}{
		{"cancelado", []string{"pending", "cancelled"}, 0, "se canceló"},
		{"caducado", []string{"expired"}, 0, "caducó"},
		{"secreto desconocido", nil, http.StatusNotFound, "caducó"},
		{"estado raro", []string{"quien-sabe"}, 0, "estado desconocido"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newFakeServer(t, tc.polls...)
			srv.pollCode = tc.pollCode
			h := newHarness(t)
			if code := h.run("login", "--servidor", srv.srv.URL); code != 1 || !strings.Contains(h.errOut.String(), tc.want) {
				t.Fatalf("salida %d: %s", code, h.errOut)
			}
			if _, err := os.Stat(credentialsPath(h.app.Home)); !os.IsNotExist(err) {
				t.Fatal("no debe guardar credenciales")
			}
		})
	}
}

func TestLoginReintentaSiFallaLaRed(t *testing.T) {
	srv := newFakeServer(t, "authorized")
	h := newHarness(t)
	failures := 2
	base := http.DefaultTransport
	h.app.HTTP = &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/cli/login/poll" && failures > 0 {
			failures--
			return nil, errors.New("conexión rechazada")
		}
		return base.RoundTrip(r)
	})}
	if code := h.run("login", "--servidor", srv.srv.URL); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if strings.Count(h.errOut.String(), "reintentando") != 1 {
		t.Fatalf("avisa una vez: %s", h.errOut)
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestServidorPorOmisionYAlternativo(t *testing.T) {
	var asked string
	h := newHarness(t)
	h.app.HTTP = &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		asked = r.URL.Scheme + "://" + r.URL.Host
		return nil, errors.New("sin red")
	})}
	if code := h.run("login"); code != 1 || asked != DefaultServer {
		t.Fatalf("por omisión pidió a %q (%d)", asked, code)
	}
	if code := h.run("login", "--servidor", "http://127.0.0.1:8080/"); code != 1 || asked != "http://127.0.0.1:8080" {
		t.Fatalf("alternativo pidió a %q (%d)", asked, code)
	}
	for _, bad := range []string{"pmt.ejemplo.com", "ftp://x", "http://", "https://x/?a=1"} {
		h.errOut.Reset()
		if code := h.run("login", "--servidor", bad); code != 1 || !strings.Contains(h.errOut.String(), "servidor inválido") {
			t.Errorf("%q: %d %s", bad, code, h.errOut)
		}
	}
}

func TestCredencialesConPermisos(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "pmt", "tokens"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := Credentials{Server: "https://pmt.example.com", Username: "saul", Token: "pmt_x", TokenID: "id"}
	if err := saveCredentials(home, c); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(credentialsPath(home))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("archivo: %v %v", info.Mode(), err)
	}
	dir, err := os.Stat(configDir(home))
	if err != nil || dir.Mode().Perm() != 0o700 {
		t.Fatalf("carpeta: %v %v", dir.Mode(), err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "pmt", "tokens")); err != nil {
		t.Fatal("no toca tokens/ del kit")
	}
	raw, _ := os.ReadFile(credentialsPath(home))
	for _, key := range []string{"servidor:", "username:", "token:", "token_id:"} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("falta %s en %s", key, raw)
		}
	}
	got, err := loadCredentials(home)
	if err != nil || got != c {
		t.Fatalf("leídas = %+v, %v", got, err)
	}
	entries, _ := os.ReadDir(configDir(home))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".credenciales-") {
			t.Fatalf("quedó un temporal: %s", e.Name())
		}
	}
	if err := removeCredentials(home); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCredentials(home); !errors.Is(err, errNoCredentials) {
		t.Fatalf("sin archivo: %v", err)
	}
}

func TestWhoamiConTokenRevocadoYLogoutSinConexion(t *testing.T) {
	srv := newFakeServer(t)
	srv.meStatus = http.StatusUnauthorized
	h := newHarness(t)
	if err := saveCredentials(h.app.Home, Credentials{Server: srv.srv.URL, Username: "saul", Token: "pmt_viejo", TokenID: "id"}); err != nil {
		t.Fatal(err)
	}
	if code := h.run("whoami"); code != 1 || !strings.Contains(h.errOut.String(), "ya no es válida") {
		t.Fatalf("whoami %d: %s", code, h.errOut)
	}
	srv.srv.Close()
	h.errOut.Reset()
	if code := h.run("logout"); code != 1 || !strings.Contains(h.errOut.String(), "la sesión sigue guardada") {
		t.Fatalf("logout sin conexión %d: %s", code, h.errOut)
	}
	if _, err := loadCredentials(h.app.Home); err != nil {
		t.Fatalf("sin conexión no borra: %v", err)
	}
}

func TestUsoYVersion(t *testing.T) {
	h := newHarness(t)
	if code := h.run(); code != 2 {
		t.Fatalf("sin argumentos: %d", code)
	}
	if code := h.run("volar"); code != 2 || !strings.Contains(h.errOut.String(), "comando desconocido") {
		t.Fatalf("desconocido: %d", code)
	}
	if code := h.run("whoami", "extra"); code != 2 {
		t.Fatalf("whoami con argumentos: %d", code)
	}
	if code := h.run("login", "--nada"); code != 2 {
		t.Fatalf("opción desconocida: %d", code)
	}
	if code := h.run("version"); code != 0 || !strings.HasPrefix(h.out.String(), "pmt dev (") {
		t.Fatalf("version: %d %q", code, h.out)
	}
}
