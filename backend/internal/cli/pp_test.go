package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const tokenPrueba = "pmt_" + "secreto000000000000000000000000000000000000000000000000000000000"

func TestFindRoot(t *testing.T) {
	base := t.TempDir()
	deep := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findRoot(deep); got != deep {
		t.Fatalf("sin .pmt la raíz es la carpeta actual: %s", got)
	}
	if err := os.Mkdir(filepath.Join(base, "a", pmtDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findRoot(deep); got != filepath.Join(base, "a") {
		t.Fatalf("raíz = %s", got)
	}
	// Un archivo .pmt no cuenta.
	if err := os.WriteFile(filepath.Join(base, "a", "b", pmtDir), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := findRoot(deep); got != filepath.Join(base, "a") {
		t.Fatalf("raíz con archivo .pmt = %s", got)
	}
}

func TestConfigIdaYVuelta(t *testing.T) {
	root := t.TempDir()
	if _, err := loadConfig(root); !errors.Is(err, errNoConfig) {
		t.Fatalf("sin archivo: %v", err)
	}
	since := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	cfg := Config{
		Proyecto:  ProjectConfig{NuevoDesde: &since},
		Base:      DatabaseConfig{Consulta: `psql -c "<consulta>"`},
		Puertos:   PortsConfig{Backend: 8090},
		Herdr:     HerdrConfig{Workspace: "demo"},
		CodexHome: "~/.codex-flow",
		MCP:       map[string]MCPConfig{"dbp": {URL: "https://dbp.ejemplo.com/mcp", Token: "tokens/dbp"}},
		Perfiles:  map[string]ProfileConfig{"pp": {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high", MCP: []string{"dbp"}}},
	}
	if err := saveConfig(root, cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(configPath(root))
	for _, want := range []string{"# Configuración de pmt", "proyecto:", "nuevo_desde:", "codex_home: ~/.codex-flow", "perfiles:", "herramienta: codex"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("falta %q en:\n%s", want, raw)
		}
	}
	got, err := loadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Version = 1
	if fmt.Sprint(got) != fmt.Sprint(cfg) || !got.Proyecto.NuevoDesde.Equal(since) {
		t.Fatalf("leído = %+v\nesperado = %+v", got, cfg)
	}
	if err := os.WriteFile(configPath(root), []byte("proyecto: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(root); err == nil || !strings.Contains(err.Error(), "dañado") {
		t.Fatalf("dañado: %v", err)
	}
}

func TestEnsureGitignoreSinDuplicar(t *testing.T) {
	root := t.TempDir()
	changed, err := ensureGitignore(root)
	if err != nil || !changed {
		t.Fatalf("crear: %v %v", changed, err)
	}
	if changed, _ := ensureGitignore(root); changed {
		t.Fatal("no debe duplicar")
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if string(raw) != ".pmt/\n" {
		t.Fatalf(".gitignore = %q", raw)
	}

	for _, existing := range []string{"node_modules\n/.pmt\n", "dist\n.pmt", "  .pmt/  \n"} {
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(existing), 0o644); err != nil {
			t.Fatal(err)
		}
		if changed, _ := ensureGitignore(root); changed {
			t.Fatalf("%q ya lo tenía", existing)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("dist"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, _ := ensureGitignore(root); !changed {
		t.Fatal("debía agregarlo")
	}
	raw, _ = os.ReadFile(filepath.Join(root, ".gitignore"))
	if string(raw) != "dist\n.pmt/\n" {
		t.Fatalf(".gitignore = %q", raw)
	}
}

func TestRenderEntornoPendiente(t *testing.T) {
	cfg := defaultConfig("/p/demo")
	got := renderEntorno(cfg, "/p/demo", "ana")
	for _, want := range []string{
		"# Entorno del proyecto",
		"## Proyecto\n- Ruta: `/p/demo`\n- Descripción: " + pending,
		"- Proyecto en PMT: todavía no existe; lo crea PP",
		"- Cadena: " + pending,
		"- Consultar: " + pending,
		"- Backend: " + pending,
		"- Frontend: " + pending,
		"- Avisar: `pmt avisar <agente> \"mensaje\"`",
		"- Levantar: `pmt levantar <agente> \"encargo\"`",
		"- Reiniciar: `pmt reiniciar <agente>`",
		"- Relanzar a DP: `pmt relanzar-dp <medium|high>`",
		"## Para el usuario\n- Archivo: `/p/demo/.pmt/para-usuario.md`",
		"| Agente | PMT | Perfil |\n|---|---|---|\n| pp | ana | PP |\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("falta %q en:\n%s", want, got)
		}
	}
	// Las secciones van en el orden de .docs/perfiles.md.
	last := -1
	for _, h := range []string{"## Proyecto", "## Base de datos", "## Puertos", "## Comandos del equipo", "## Para el usuario", "## Agentes"} {
		i := strings.Index(got, h)
		if i < last {
			t.Fatalf("%s fuera de orden", h)
		}
		last = i
	}
}

func TestRenderEntornoCompleto(t *testing.T) {
	cfg := Config{
		Proyecto: ProjectConfig{ID: "id-1", Nombre: "Demo", Ruta: "/srv/demo", Descripcion: "Tiene backend/ (Go)."},
		Base:     DatabaseConfig{Cadena: "postgres://u@localhost/demo", Consulta: `psql -d demo -c "<consulta>"`},
		Puertos:  PortsConfig{Backend: 8090, Frontend: 5173},
		Herdr:    HerdrConfig{Workspace: "demo", Prefijo: "demo-"},
		Perfiles: map[string]ProfileConfig{
			"pp":         {Herramienta: "claude", Modelo: "opus", Effort: "medium"},
			"devbackend": {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"},
			"tl":         {Herramienta: "claude", Modelo: "opus", Effort: "medium"},
		},
	}
	got := renderEntorno(cfg, "/p/demo", "ana")
	for _, want := range []string{
		"- Ruta: `/srv/demo`\n- Descripción: Tiene backend/ (Go).\n- Proyecto en PMT: Demo (`id-1`)",
		"- Cadena: `postgres://u@localhost/demo` (solo DevBackend",
		"- Consultar: `psql -d demo -c \"<consulta>\"` (cambia `<consulta>` por la tuya)",
		"- Backend: 8090\n- Frontend: 5173",
		"| demo-pp | ana | PP |\n| demo-tl | ana | TL |\n| demo-devback | ana | DevBackend |\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("falta %q en:\n%s", want, got)
		}
	}
	if strings.Contains(got, "pendiente") {
		t.Fatalf("no debe quedar nada pendiente:\n%s", got)
	}
}

func launchConfig(tool string) Config {
	return Config{
		Herdr:    HerdrConfig{Workspace: "demo"},
		MCP:      map[string]MCPConfig{"dbp": {URL: "https://dbp.ejemplo.com/mcp", Token: "tokens/dbp"}},
		Perfiles: map[string]ProfileConfig{"pp": {Herramienta: tool, Modelo: defaultModel[tool], Effort: "high", MCP: []string{"dbp"}}},
	}
}

func writeDBPToken(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(configDir(home), "tokens")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dbp"), []byte("dbp-secreto\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildLaunchClaude(t *testing.T) {
	home := t.TempDir()
	writeDBPToken(t, home)
	spec, err := buildLaunch(launchInput{
		Config: launchConfig("claude"), Root: "/p/demo", Profile: "pp", Server: "https://pmt.ejemplo.com",
		Token: tokenPrueba, ProfileFile: "/p/demo/.pmt/perfiles/pp.md", MCPFile: "/p/demo/.pmt/mcp-pp.json", Home: home,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--dangerously-skip-permissions", "--autocompact", "400k", "--model", "opus", "--effort", "high",
		"--mcp-config", "/p/demo/.pmt/mcp-pp.json", "--strict-mcp-config", "--append-system-prompt-file", "/p/demo/.pmt/perfiles/pp.md"}
	if spec.Tool != "claude" || strings.Join(spec.Args, " ") != strings.Join(want, " ") {
		t.Fatalf("args = %q", spec.Args)
	}
	if spec.Env[tokenEnv] != tokenPrueba || spec.Env["MCP_TOKEN_DBP"] != "dbp-secreto" {
		t.Fatalf("env = %v", spec.Env)
	}
	assertNoSecrets(t, strings.Join(spec.Args, " "))
	if strings.Contains(strings.Join(spec.Args, " "), "engram") {
		t.Fatal("el plugin engram es de esta máquina")
	}

	raw, err := mcpConfigJSON(launchConfig("claude"), "https://pmt.ejemplo.com", "pp")
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecrets(t, string(raw))
	var doc struct {
		MCPServers map[string]struct {
			Type    string            `json:"type"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	pmt, dbp := doc.MCPServers["pmt"], doc.MCPServers["dbp"]
	if pmt.URL != "https://pmt.ejemplo.com/mcp?profile=pp" || pmt.Type != "http" || pmt.Headers["Authorization"] != "Bearer ${PMT_TOKEN:-}" {
		t.Fatalf("pmt = %+v", pmt)
	}
	if dbp.URL != "https://dbp.ejemplo.com/mcp" || dbp.Headers["Authorization"] != "Bearer ${MCP_TOKEN_DBP:-}" {
		t.Fatalf("dbp = %+v", dbp)
	}
}

func TestBuildLaunchCodex(t *testing.T) {
	home := t.TempDir()
	writeDBPToken(t, home)
	cfg := launchConfig("codex")
	spec, err := buildLaunch(launchInput{
		Config: cfg, Root: "/p/demo", Profile: "pp", Server: "https://pmt.ejemplo.com",
		Token: tokenPrueba, ProfileText: "# PP\nUsa \"comillas\".\n", Home: home,
	})
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(spec.Args, "\n")
	for _, want := range []string{
		"-m\ngpt-6.1-terra",
		`developer_instructions="# PP\nUsa \"comillas\".\n"`,
		`model_reasoning_effort="high"`,
		`approval_policy="never"`,
		`sandbox_mode="danger-full-access"`,
		"allow_login_shell=false",
		"notice.hide_rate_limit_model_nudge=true",
		`mcp_servers.pmt.url="https://pmt.ejemplo.com/mcp?profile=pp"`,
		`mcp_servers.pmt.bearer_token_env_var="PMT_TOKEN"`,
		`projects={"/p/demo"={trust_level="trusted"}}`,
		`mcp_servers.dbp.url="https://dbp.ejemplo.com/mcp"`,
		`mcp_servers.dbp.bearer_token_env_var="MCP_TOKEN_DBP"`,
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("falta %q en:\n%s", want, args)
		}
	}
	assertNoSecrets(t, args)
	if spec.Env["CODEX_HOME"] != filepath.Join(home, ".codex") {
		t.Fatalf("CODEX_HOME = %q", spec.Env["CODEX_HOME"])
	}
	cfg.CodexHome = "~/.codex-flow"
	if got := cfg.codexHome(home); got != filepath.Join(home, ".codex-flow") {
		t.Fatalf("codex_home = %q", got)
	}
	cfg.CodexHome = "/opt/codex"
	if got := cfg.codexHome(home); got != "/opt/codex" {
		t.Fatalf("codex_home = %q", got)
	}
}

func TestBuildLaunchRechazaMCPMal(t *testing.T) {
	home := t.TempDir()
	for name, mod := range map[string]func(*Config){
		"sin token":    func(c *Config) {},
		"no declarado": func(c *Config) { delete(c.MCP, "dbp") },
		"nombre inválido": func(c *Config) {
			c.Perfiles["pp"] = ProfileConfig{Herramienta: "claude", Modelo: "opus", MCP: []string{"DBP-x"}}
		},
		"ruta absoluta": func(c *Config) { c.MCP["dbp"] = MCPConfig{URL: "https://x/mcp", Token: "/etc/passwd"} },
		"sale de la carpeta": func(c *Config) {
			c.MCP["dbp"] = MCPConfig{URL: "https://x/mcp", Token: "../secreto"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := launchConfig("claude")
			mod(&cfg)
			if _, err := buildLaunch(launchInput{Config: cfg, Profile: "pp", Home: home}); err == nil {
				t.Fatal("debía fallar")
			}
		})
	}
	cfg := launchConfig("qwen")
	if _, err := buildLaunch(launchInput{Config: cfg, Profile: "pp", Home: home}); err == nil {
		t.Fatal("solo claude y codex")
	}
}

func TestEnvListReemplaza(t *testing.T) {
	got := envList([]string{"PATH=/bin", "PMT_TOKEN=viejo", "HOME=/h"}, map[string]string{"PMT_TOKEN": "nuevo", "A": "1"})
	if strings.Join(got, " ") != "PATH=/bin HOME=/h A=1 PMT_TOKEN=nuevo" {
		t.Fatalf("env = %v", got)
	}
}

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"/usr/bin/pmt":  "/usr/bin/pmt",
		"/home/a b/pmt": "'/home/a b/pmt'",
		"/x/it's":       `'/x/it'\''s'`,
		"":              "''",
	} {
		if got := shellQuote(in); got != want {
			t.Fatalf("shellQuote(%q) = %q", in, got)
		}
	}
}

func TestNewProjectCandidates(t *testing.T) {
	since := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	got := newProjectCandidates([]projectItem{
		{ID: "viejo", Role: "owner", CreatedAt: since.Add(-time.Minute)},
		{ID: "nuevo", Role: "owner", CreatedAt: since.Add(time.Minute)},
		{ID: "compartido", Role: "editor", CreatedAt: since.Add(time.Minute)},
		{ID: "justo", Role: "owner", CreatedAt: since},
	}, since)
	if len(got) != 2 || got[0].ID != "nuevo" || got[1].ID != "justo" {
		t.Fatalf("candidatos = %+v", got)
	}
}

func assertNoSecrets(t *testing.T, s string) {
	t.Helper()
	if strings.Contains(s, tokenPrueba) || strings.Contains(s, "dbp-secreto") {
		t.Fatalf("hay un token en: %s", s)
	}
}

// --- pmt pp de punta a punta con herdr y servidor falsos ---

type fakeHerdr struct {
	mu         sync.Mutex
	workspaces []herdrWorkspace
	tabs       []herdrTab
	panes      []herdrPane
	agents     []herdrAgent
	busy       map[string]bool
	runs       []string
	renamed    map[string]string
	focused    []string
	next       int
	// Para los comandos del equipo.
	screen  map[string]string // lo visible de cada pane
	keys    []string          // pane: teclas
	texts   []string          // pane: texto
	prompts []string          // pane: mensaje
	// hook corre tras cada Run, Prompt, SendKeys y SendText, para simular al agente.
	hook func(f *fakeHerdr, op, pane, arg string)
}

func (f *fakeHerdr) id(prefix string) string { f.next++; return fmt.Sprintf("%s%d", prefix, f.next) }

func (f *fakeHerdr) Workspaces(context.Context) ([]herdrWorkspace, error) { return f.workspaces, nil }
func (f *fakeHerdr) CreateWorkspace(_ context.Context, _, label string) (herdrWorkspace, herdrTab, string, error) {
	w := herdrWorkspace{ID: f.id("w"), Label: label}
	f.workspaces = append(f.workspaces, w)
	tab, pane, _ := f.CreateTab(context.Background(), w.ID, "", "1")
	return w, tab, pane, nil
}
func (f *fakeHerdr) Tabs(_ context.Context, ws string) ([]herdrTab, error) {
	var out []herdrTab
	for _, t := range f.tabs {
		if t.Workspace == ws {
			out = append(out, t)
		}
	}
	return out, nil
}
func (f *fakeHerdr) CreateTab(_ context.Context, ws, _, label string) (herdrTab, string, error) {
	t := herdrTab{ID: f.id("t"), Label: label, Workspace: ws}
	p := herdrPane{ID: f.id("p"), Tab: t.ID, Workspace: ws}
	f.tabs, f.panes = append(f.tabs, t), append(f.panes, p)
	return t, p.ID, nil
}
func (f *fakeHerdr) RenameTab(_ context.Context, tab, label string) error {
	for i := range f.tabs {
		if f.tabs[i].ID == tab {
			f.tabs[i].Label = label
		}
	}
	return nil
}
func (f *fakeHerdr) FocusTab(_ context.Context, tab string) error {
	f.focused = append(f.focused, tab)
	return nil
}
func (f *fakeHerdr) Panes(_ context.Context, ws string) ([]herdrPane, error) {
	var out []herdrPane
	for _, p := range f.panes {
		if p.Workspace == ws {
			out = append(out, p)
		}
	}
	return out, nil
}
func (f *fakeHerdr) Busy(_ context.Context, pane string) (bool, error) { return f.busy[pane], nil }
func (f *fakeHerdr) Run(_ context.Context, pane, command string) error {
	f.runs = append(f.runs, pane+": "+command)
	for i := range f.panes {
		if f.panes[i].ID != pane {
			continue
		}
		switch {
		case command == "/exit" || command == "/quit":
			// El agente sale y queda el shell.
			f.panes[i].Agent, f.panes[i].Status = "", ""
			f.busy[pane] = false
		case strings.HasPrefix(command, "/"):
		default:
			// El agente arranca en el pane.
			f.panes[i].Agent, f.panes[i].Status = "claude", "idle"
			if f.busy != nil {
				f.busy[pane] = true
			}
		}
	}
	f.fire("run", pane, command)
	return nil
}

// stop simula que el agente del pane terminó: queda el shell.
func (f *fakeHerdr) stop(pane string) {
	if p := f.pane(pane); p != nil {
		p.Agent, p.Status = "", ""
	}
	f.busy[pane] = false
}
func (f *fakeHerdr) fire(op, pane, arg string) {
	if f.hook != nil {
		f.hook(f, op, pane, arg)
	}
}
func (f *fakeHerdr) pane(id string) *herdrPane {
	for i := range f.panes {
		if f.panes[i].ID == id {
			return &f.panes[i]
		}
	}
	return nil
}
func (f *fakeHerdr) Pane(_ context.Context, pane string) (herdrPane, error) {
	if p := f.pane(pane); p != nil {
		return *p, nil
	}
	return herdrPane{}, fmt.Errorf("pane %s no existe", pane)
}
func (f *fakeHerdr) Read(_ context.Context, pane string, _ int) (string, error) {
	return f.screen[pane], nil
}
func (f *fakeHerdr) SendKeys(_ context.Context, pane string, keys ...string) error {
	k := strings.Join(keys, " ")
	f.keys = append(f.keys, pane+": "+k)
	f.fire("keys", pane, k)
	return nil
}
func (f *fakeHerdr) SendText(_ context.Context, pane, text string) error {
	f.texts = append(f.texts, pane+": "+text)
	f.fire("text", pane, text)
	return nil
}
func (f *fakeHerdr) Prompt(_ context.Context, pane, text string) error {
	f.prompts = append(f.prompts, pane+": "+text)
	f.fire("prompt", pane, text)
	return nil
}
func (f *fakeHerdr) Agents(context.Context) ([]herdrAgent, error) { return f.agents, nil }
func (f *fakeHerdr) RenameAgent(_ context.Context, pane, name string) error {
	if f.renamed == nil {
		f.renamed = map[string]string{}
	}
	f.renamed[pane] = name
	return nil
}

type ppServer struct {
	srv          *httptest.Server
	projects     []projectItem
	profileCode  int
	profileCalls int
	description  string // de GET /api/projects/{id}
}

func newPPServer(t *testing.T) *ppServer {
	s := &ppServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tokenPrueba {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(s.projects)
	})
	mux.HandleFunc("GET /api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"id": r.PathValue("id"), "description": s.description})
	})
	mux.HandleFunc("GET /api/profiles/{p}", func(w http.ResponseWriter, r *http.Request) {
		s.profileCalls++
		if s.profileCode != 0 {
			w.WriteHeader(s.profileCode)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		fmt.Fprintf(w, "# Perfil %s\n", r.PathValue("p"))
	})
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

type ppHarness struct {
	*harness
	root    string
	hd      *fakeHerdr
	env     map[string]string
	execs   [][]string
	execEnv []string
}

func newPPHarness(t *testing.T, srv *ppServer) *ppHarness {
	h := &ppHarness{harness: newHarness(t), root: filepath.Join(t.TempDir(), "demo"), hd: &fakeHerdr{busy: map[string]bool{}, screen: map[string]string{}},
		env: map[string]string{"HERDR_ENV": "1"}}
	if err := os.MkdirAll(filepath.Join(h.root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := saveCredentials(h.app.Home, Credentials{Server: srv.srv.URL, Username: "ana", Token: tokenPrueba, TokenID: "id"}); err != nil {
		t.Fatal(err)
	}
	// Enter en las cuatro preguntas de la primera vez: proyecto, herramienta,
	// modelo y effort. Sin respuestas, la pregunta corta (Ctrl+D).
	h.app.Stdin = strings.NewReader("\n\n\n\n")
	h.app.Getenv = func(k string) string { return h.env[k] }
	h.app.Environ = func() []string { return []string{"PATH=/bin"} }
	h.app.Getwd = func() (string, error) { return h.root, nil }
	h.app.LookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	h.app.Executable = func() (string, error) { return "/opt/mis cosas/pmt", nil }
	h.app.Herdr = func() (herdr, error) { return h.hd, nil }
	h.app.Exec = func(bin string, argv, env []string) error {
		h.execs = append(h.execs, append([]string{bin}, argv...))
		h.execEnv = env
		return nil
	}
	return h
}

func TestPPPrimeraVezConfiguraYLanza(t *testing.T) {
	srv := newPPServer(t)
	srv.projects = []projectItem{
		{ID: "id-dbp", Name: "DBP", Role: "owner"},
		{ID: "id-demo", Name: "Demo", Role: "editor"},
		{ID: "id-ver", Name: "Solo ver", Role: "viewer"},
	}
	h := newPPHarness(t, srv)
	// Proyecto: Enter (Demo, que se llama como la carpeta); herramienta, modelo y effort por omisión.
	h.app.Stdin = strings.NewReader("\n\n\nhigh\n")
	if code := h.run("pp"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	out := h.out.String()
	if strings.Contains(out, "Solo ver") || !strings.Contains(out, "2) Demo (editor)") || !strings.Contains(out, "3) Nuevo: lo crea PP") || !strings.Contains(out, "Elige [2]") {
		t.Fatalf("lista de proyectos:\n%s", out)
	}
	cfg, err := loadConfig(h.root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Proyecto.ID != "id-demo" || cfg.Herdr.Workspace != "demo" || fmt.Sprint(cfg.Perfiles["pp"]) != fmt.Sprint(ProfileConfig{Herramienta: "claude", Modelo: "opus", Effort: "high"}) {
		t.Fatalf("config = %+v", cfg)
	}
	gi, _ := os.ReadFile(filepath.Join(h.root, ".gitignore"))
	if string(gi) != ".pmt/\n" {
		t.Fatalf(".gitignore = %q", gi)
	}
	entorno, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "entorno.md"))
	if !strings.Contains(string(entorno), "| pp | ana | PP |") || !strings.Contains(string(entorno), "Demo (`id-demo`)") {
		t.Fatalf("entorno:\n%s", entorno)
	}
	perfil, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "perfiles", "pp.md"))
	if string(perfil) != "# Perfil pp\n" {
		t.Fatalf("perfil = %q", perfil)
	}
	if _, err := os.Stat(filepath.Join(h.root, pmtDir, "para-usuario.md")); err != nil {
		t.Fatal(err)
	}
	mcp, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "mcp-pp.json"))
	if !strings.Contains(string(mcp), srv.srv.URL+"/mcp?profile=pp") {
		t.Fatalf("mcp = %s", mcp)
	}
	// .pmt/ nunca lleva el token.
	filepath.Walk(filepath.Join(h.root, pmtDir), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			raw, _ := os.ReadFile(p)
			assertNoSecrets(t, string(raw))
		}
		return nil
	})

	// herdr: workspace nuevo, su primera pestaña renombrada a pp y el lanzador sin token.
	if len(h.hd.workspaces) != 1 || h.hd.workspaces[0].Label != "demo" || h.hd.tabs[0].Label != "pp" {
		t.Fatalf("herdr = %+v %+v", h.hd.workspaces, h.hd.tabs)
	}
	pane := h.hd.panes[0].ID
	if len(h.hd.runs) != 1 || h.hd.runs[0] != pane+": '/opt/mis cosas/pmt' _lanzar pp "+h.root {
		t.Fatalf("runs = %q", h.hd.runs)
	}
	assertNoSecrets(t, strings.Join(h.hd.runs, " ")+out+h.errOut.String())
	if h.hd.renamed[pane] != "pp" || len(h.hd.focused) != 1 || h.hd.focused[0] != h.hd.tabs[0].ID {
		t.Fatalf("renombrado %v, enfocado %v", h.hd.renamed, h.hd.focused)
	}

	// El lanzador, como correría en el pane: token solo en el entorno.
	if code := h.run(launcherCommand, "pp", h.root); code != 0 {
		t.Fatalf("lanzador %d: %s", code, h.errOut)
	}
	if len(h.execs) != 1 || h.execs[0][0] != "/usr/bin/claude" || h.execs[0][1] != "claude" {
		t.Fatalf("exec = %q", h.execs)
	}
	assertNoSecrets(t, strings.Join(h.execs[0], " "))
	if !strings.Contains(strings.Join(h.execEnv, "\n"), "PMT_TOKEN="+tokenPrueba) {
		t.Fatal("el token va en el entorno del agente")
	}

	// Segunda vez: PP ya corre, no pregunta ni relanza.
	h.hd.agents = []herdrAgent{{Name: "pp", Pane: pane, Workspace: h.hd.workspaces[0].ID, Agent: "claude"}}
	h.out.Reset()
	if code := h.run("pp"); code != 0 {
		t.Fatalf("segunda %d: %s", code, h.errOut)
	}
	if len(h.hd.runs) != 1 || !strings.Contains(h.out.String(), "PP ya corre") || len(h.hd.focused) != 2 {
		t.Fatalf("relanzó: %q\n%s", h.hd.runs, h.out)
	}
}

func TestPPRequisitos(t *testing.T) {
	srv := newPPServer(t)
	h := newPPHarness(t, srv)
	h.env["HERDR_ENV"] = ""
	if code := h.run("pp"); code != 1 || !strings.Contains(h.errOut.String(), "dentro de herdr") {
		t.Fatalf("sin herdr %d: %s", code, h.errOut)
	}
	h.env["HERDR_ENV"] = "1"
	if err := removeCredentials(h.app.Home); err != nil {
		t.Fatal(err)
	}
	h.errOut.Reset()
	if code := h.run("pp"); code != 1 || !strings.Contains(h.errOut.String(), "pmt login") {
		t.Fatalf("sin login %d: %s", code, h.errOut)
	}
	if err := saveCredentials(h.app.Home, Credentials{Server: srv.srv.URL, Username: "ana", Token: tokenPrueba}); err != nil {
		t.Fatal(err)
	}
	h.app.LookPath = func(string) (string, error) { return "", errors.New("no") }
	h.errOut.Reset()
	if code := h.run("pp"); code != 1 || !strings.Contains(h.errOut.String(), "no encuentro claude") {
		t.Fatalf("sin claude %d: %s", code, h.errOut)
	}
	if len(h.hd.runs) != 0 {
		t.Fatal("no debía lanzar")
	}
}

func TestPPProyectoNuevoYReconfigurar(t *testing.T) {
	srv := newPPServer(t)
	srv.projects = []projectItem{{ID: "id-dbp", Name: "DBP", Role: "owner", CreatedAt: time.Now().Add(-time.Hour)}}
	h := newPPHarness(t, srv)
	start := time.Now().UTC().Truncate(time.Second)
	h.app.Now = func() time.Time { return start }
	// Nuevo, codex, modelo y effort por omisión.
	h.app.Stdin = strings.NewReader("2\ncodex\n\n\n")
	if code := h.run("pp"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	cfg, _ := loadConfig(h.root)
	if cfg.Proyecto.ID != "" || cfg.Proyecto.NuevoDesde == nil || cfg.Perfiles["pp"].Modelo != "gpt-6.1-terra" {
		t.Fatalf("config = %+v", cfg)
	}
	entorno, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "entorno.md"))
	if !strings.Contains(string(entorno), "lo crea PP") {
		t.Fatalf("entorno:\n%s", entorno)
	}
	if _, err := os.Stat(filepath.Join(h.root, pmtDir, "mcp-pp.json")); !os.IsNotExist(err) {
		t.Fatal("Codex no usa mcp-pp.json")
	}

	// PP creó el proyecto; en el siguiente arranque se confirma y se guarda.
	srv.projects = append(srv.projects, projectItem{ID: "id-nuevo", Name: "Demo", Role: "owner", CreatedAt: start.Add(time.Minute)})
	h.hd.stop(h.hd.panes[0].ID) // PP terminó.
	h.app.Stdin = strings.NewReader("\n")
	if code := h.run("pp"); code != 0 {
		t.Fatalf("segunda %d: %s", code, h.errOut)
	}
	cfg, _ = loadConfig(h.root)
	if cfg.Proyecto.ID != "id-nuevo" || cfg.Proyecto.NuevoDesde != nil || !strings.Contains(h.out.String(), "PP creó el proyecto «Demo»") {
		t.Fatalf("config = %+v\n%s", cfg, h.out)
	}

	// --configurar propone lo actual: Enter en todo lo deja igual salvo la herramienta.
	h.hd.stop(h.hd.panes[0].ID)
	h.out.Reset()
	h.app.Stdin = strings.NewReader("\nclaude\n\n\n")
	if code := h.run("pp", "--configurar"); code != 0 {
		t.Fatalf("configurar %d: %s", code, h.errOut)
	}
	if !strings.Contains(h.out.String(), "Elige [2]") || !strings.Contains(h.out.String(), "Herramienta de PP (claude o codex) [codex]") || !strings.Contains(h.out.String(), "Modelo [opus]") {
		t.Fatalf("omisiones:\n%s", h.out)
	}
	cfg, _ = loadConfig(h.root)
	if cfg.Proyecto.ID != "id-nuevo" || cfg.Perfiles["pp"].Herramienta != "claude" || cfg.Perfiles["pp"].Modelo != "opus" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestPPPerfilSinServidorUsaCopia(t *testing.T) {
	srv := newPPServer(t)
	h := newPPHarness(t, srv)
	if code := h.run("pp"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	srv.profileCode = http.StatusBadGateway
	h.hd.stop(h.hd.panes[0].ID)
	if code := h.run("pp"); code != 0 || !strings.Contains(h.errOut.String(), "uso la copia") {
		t.Fatalf("con copia %d: %s", code, h.errOut)
	}
	if err := os.Remove(filepath.Join(h.root, pmtDir, "perfiles", "pp.md")); err != nil {
		t.Fatal(err)
	}
	h.hd.stop(h.hd.panes[0].ID)
	h.errOut.Reset()
	if code := h.run("pp"); code != 1 || !strings.Contains(h.errOut.String(), "no hay copia local") {
		t.Fatalf("sin copia %d: %s", code, h.errOut)
	}
}

func TestPPPrefijoSiOtroEquipoTienePP(t *testing.T) {
	srv := newPPServer(t)
	h := newPPHarness(t, srv)
	h.hd.agents = []herdrAgent{{Name: "pp", Pane: "otro:p1", Workspace: "otro"}}
	if code := h.run("pp"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	cfg, _ := loadConfig(h.root)
	if cfg.Herdr.Prefijo != "demo-" || h.hd.renamed[h.hd.panes[0].ID] != "demo-pp" {
		t.Fatalf("prefijo %q, nombres %v", cfg.Herdr.Prefijo, h.hd.renamed)
	}
	entorno, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "entorno.md"))
	if !strings.Contains(string(entorno), "| demo-pp | ana | PP |") {
		t.Fatalf("entorno:\n%s", entorno)
	}
}

// --- config.toml de Codex con entradas stdio ---

func codexLaunchWithToml(t *testing.T, toml string) (launchSpec, string, error) {
	t.Helper()
	home := t.TempDir()
	writeDBPToken(t, home)
	if toml != "" {
		dir := filepath.Join(home, ".codex")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(toml), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	spec, err := buildLaunch(launchInput{
		Config: launchConfig("codex"), Root: "/p/demo", Profile: "pp", Server: "https://pmt.ejemplo.com",
		Token: tokenPrueba, ProfileText: "# PP\n", Home: home,
	})
	return spec, home, err
}

func TestBuildLaunchCodexConfigToml(t *testing.T) {
	const pmtStdio = "[mcp_servers.pmt]\ncommand = \"/opt/pmt-v3/mcp\"\nargs = [\"stdio\"]\n[mcp_servers.pmt.env]\nDB = \"x\"\n"
	cases := []struct {
		name     string
		toml     string
		want     []string
		missing  []string
		warnings int
	}{
		{
			name:    "sin config.toml",
			want:    []string{`mcp_servers.pmt.url="https://pmt.ejemplo.com/mcp?profile=pp"`, `mcp_servers.dbp.url=`},
			missing: []string{"enabled=false", "_cli"},
		},
		{
			name:    "pmt de tipo url",
			toml:    "model = \"x\"\n[mcp_servers.pmt]\nurl = \"http://localhost:8080/mcp\"\n",
			want:    []string{`mcp_servers.pmt.url="https://pmt.ejemplo.com/mcp?profile=pp"`},
			missing: []string{"enabled=false", "_cli"},
		},
		{
			name: "pmt stdio",
			toml: pmtStdio,
			want: []string{
				"mcp_servers.pmt.enabled=false",
				`mcp_servers.pmt_cli.url="https://pmt.ejemplo.com/mcp?profile=pp"`,
				`mcp_servers.pmt_cli.bearer_token_env_var="PMT_TOKEN"`,
				"el MCP de PMT se llama `pmt_cli`",
				`mcp_servers.dbp.url="https://dbp.ejemplo.com/mcp"`,
			},
			missing:  []string{"mcp_servers.pmt.url=", "dbp.enabled", "dbp_cli"},
			warnings: 1,
		},
		{
			name: "extra que choca",
			toml: "[mcp_servers.dbp]\ncommand = \"dbp-mcp\"\n[mcp_servers.otro]\ncommand = \"x\"\n",
			want: []string{
				`mcp_servers.pmt.url="https://pmt.ejemplo.com/mcp?profile=pp"`,
				"mcp_servers.dbp.enabled=false",
				`mcp_servers.dbp_cli.url="https://dbp.ejemplo.com/mcp"`,
				`mcp_servers.dbp_cli.bearer_token_env_var="MCP_TOKEN_DBP"`,
			},
			missing:  []string{"mcp_servers.pmt.enabled", "pmt_cli", "mcp_servers.dbp.url=", "otro"},
			warnings: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec, home, err := codexLaunchWithToml(t, c.toml)
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Join(spec.Args, "\n")
			for _, w := range c.want {
				if !strings.Contains(args, w) {
					t.Fatalf("falta %q en:\n%s", w, args)
				}
			}
			for _, m := range c.missing {
				if strings.Contains(args, m) {
					t.Fatalf("sobra %q en:\n%s", m, args)
				}
			}
			assertNoSecrets(t, args)
			if len(spec.Warnings) != c.warnings {
				t.Fatalf("avisos = %q", spec.Warnings)
			}
			for _, w := range spec.Warnings {
				if !strings.Contains(w, filepath.Join(home, ".codex", "config.toml")) || !strings.Contains(w, "codex_home") || strings.Contains(w, "\n") {
					t.Fatalf("aviso = %q", w)
				}
				assertNoSecrets(t, w)
			}
		})
	}
}

func TestBuildLaunchCodexTomlRoto(t *testing.T) {
	_, home, err := codexLaunchWithToml(t, "[mcp_servers.pmt\ncommand = \n")
	if err == nil {
		t.Fatal("debía fallar")
	}
	if !strings.Contains(err.Error(), filepath.Join(home, ".codex", "config.toml")) {
		t.Fatalf("el error no dice la ruta: %v", err)
	}
}
