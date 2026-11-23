package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const cadenaPrueba = "postgres://u:clave-secreta@localhost:5432/demo"

// teamConfig es una configuración completa con PP y los ocho del equipo.
func teamConfig() Config {
	cfg := Config{
		Proyecto: ProjectConfig{ID: "id-demo", Nombre: "Demo", Descripcion: "Tiene backend/ (Go)."},
		Base:     DatabaseConfig{Cadena: cadenaPrueba, Consulta: `psql -d demo -c "<consulta>"`},
		Puertos:  PortsConfig{Backend: 8090, Frontend: 5173},
		Herdr:    HerdrConfig{Workspace: "demo"},
		Perfiles: map[string]ProfileConfig{"pp": {Herramienta: "claude", Modelo: "opus", Effort: "medium"}},
	}
	for s, p := range proposed {
		cfg.Perfiles[s] = p
	}
	return cfg
}

// newTeamHarness deja el proyecto con config y, si up, el workspace con los
// agentes nombrados ya corriendo, cada uno en su pestaña con el compositor vacío.
func newTeamHarness(t *testing.T, cfg Config, up ...string) *ppHarness {
	t.Helper()
	srv := newPPServer(t)
	h := newPPHarness(t, srv)
	if err := saveConfig(h.root, cfg); err != nil {
		t.Fatal(err)
	}
	if err := writeEnvironment(h.root, cfg, "ana"); err != nil {
		t.Fatal(err)
	}
	if len(up) > 0 {
		ws, _, _, _ := h.hd.CreateWorkspace(context.Background(), h.root, "demo")
		h.hd.tabs[0].Label = up[0]
		for i, a := range up {
			var pane string
			if i == 0 {
				pane = h.hd.panes[0].ID
			} else {
				_, pane, _ = h.hd.CreateTab(context.Background(), ws.ID, h.root, a)
			}
			p := h.hd.pane(pane)
			info, _ := lookupProfile(a)
			p.Agent, p.Status = cfg.Perfiles[info.Session].Herramienta, "idle"
			h.hd.busy[pane] = true
			h.hd.screen[pane] = "Hola\n❯ \n"
			h.hd.agents = append(h.hd.agents, herdrAgent{Name: cfg.agentName(a), Pane: pane, Workspace: ws.ID, Agent: p.Agent})
		}
	}
	return h
}

// paneOfAgent es el pane del agente en el falso.
func (h *ppHarness) paneOfAgent(agent string) string {
	for _, a := range h.hd.agents {
		if strings.TrimPrefix(a.Name, "demo-") == agent {
			return a.Pane
		}
	}
	return ""
}

// delivers simula a un agente que recibe el aviso: lo muestra arriba del compositor.
func delivers(f *fakeHerdr, op, pane, arg string) {
	if op == "prompt" {
		f.screen[pane] = "Hola\n" + arg + "\n❯ \n"
	}
}

func TestEquipoPrimeraVezPreguntaYLanza(t *testing.T) {
	srv := newPPServer(t)
	srv.description = "Tiene backend/ (Go) y frontend/ (React)."
	h := newPPHarness(t, srv)
	cfg := Config{
		Proyecto: ProjectConfig{ID: "id-demo", Nombre: "Demo"},
		Herdr:    HerdrConfig{Workspace: "demo"},
		Perfiles: map[string]ProfileConfig{"pp": {Herramienta: "claude", Modelo: "opus", Effort: "medium"}},
	}
	if err := saveConfig(h.root, cfg); err != nil {
		t.Fatal(err)
	}
	// Descripción de PMT (Enter), cadena, consulta sin <consulta> y luego bien,
	// puertos por omisión y aceptar los perfiles de una vez.
	h.app.Stdin = strings.NewReader("\n" + cadenaPrueba + "\npsql -d demo\npsql -d demo -c \"<consulta>\"\n\n3000\n\n")
	if code := h.run("equipo"); code != 0 {
		t.Fatalf("salida %d: %s\n%s", code, h.errOut, h.out)
	}
	out := h.out.String()
	for _, want := range []string{
		"[Tiene backend/ (Go) y frontend/ (React).]",
		"Cadena de conexión de la base de datos: ",
		"debe llevar <consulta>",
		"Puerto del backend [8090]",
		"Puerto del frontend [5173]",
		"dp        claude  opus            medium",
		"devback   codex   gpt-6.1-terra   high",
		"¿Los dejo así? [S/n]",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q en:\n%s", want, out)
		}
	}
	if strings.Contains(out+h.errOut.String(), "clave-secreta") {
		t.Fatal("la cadena no se imprime")
	}
	got, _ := loadConfig(h.root)
	if got.Proyecto.Descripcion != srv.description || got.Base.Cadena != cadenaPrueba || got.Puertos.Backend != 8090 || got.Puertos.Frontend != 3000 {
		t.Fatalf("config = %+v", got)
	}
	for s, p := range proposed {
		if fmt.Sprint(got.Perfiles[s]) != fmt.Sprint(p) {
			t.Fatalf("perfil %s = %+v", s, got.Perfiles[s])
		}
	}
	if got.Proyecto.Ruta != "" {
		t.Fatal("la ruta no se pregunta ni se guarda")
	}

	// entorno.md completo, con los nueve.
	entorno, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "entorno.md"))
	if strings.Contains(string(entorno), "pendiente") {
		t.Fatalf("quedó algo pendiente:\n%s", entorno)
	}
	for _, row := range []string{"| pp | ana | PP |", "| pc | ana | PC |", "| devback | ana | DevBackend |", "| devfront | ana | DevFrontend |", "| qa | ana | QA |"} {
		if !strings.Contains(string(entorno), row) {
			t.Fatalf("falta %q en:\n%s", row, entorno)
		}
	}

	// herdr: una pestaña por agente (la primera del workspace pasa a pc), sin PP.
	if len(h.hd.tabs) != 8 || h.hd.tabs[0].Label != "pc" {
		t.Fatalf("pestañas = %+v", h.hd.tabs)
	}
	if len(h.hd.runs) != 8 {
		t.Fatalf("runs = %q", h.hd.runs)
	}
	for i, p := range teamDefault {
		info, _ := lookupProfile(p)
		want := "'/opt/mis cosas/pmt' _lanzar " + info.Session + " " + h.root
		if !strings.HasSuffix(h.hd.runs[i], want) {
			t.Fatalf("run %d = %q, quería %q", i, h.hd.runs[i], want)
		}
		if h.hd.renamed[h.hd.panes[i].ID] != p {
			t.Fatalf("nombres = %v", h.hd.renamed)
		}
		if _, err := os.Stat(filepath.Join(h.root, pmtDir, "perfiles", info.Session+".md")); err != nil {
			t.Fatalf("perfil de %s: %v", p, err)
		}
	}
	// Claude lleva mcp-<perfil>.json; Codex no.
	if _, err := os.Stat(filepath.Join(h.root, pmtDir, "mcp-tl.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(h.root, pmtDir, "mcp-qa.json")); !os.IsNotExist(err) {
		t.Fatal("Codex no usa mcp-qa.json")
	}
	assertNoSecrets(t, strings.Join(h.hd.runs, " ")+out+h.errOut.String())
	filepath.Walk(filepath.Join(h.root, pmtDir), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			raw, _ := os.ReadFile(p)
			assertNoSecrets(t, string(raw))
		}
		return nil
	})

	// Segunda vez: todos corren; no pregunta ni relanza.
	for _, p := range h.hd.panes {
		h.hd.agents = append(h.hd.agents, herdrAgent{Name: h.hd.renamed[p.ID], Pane: p.ID, Workspace: p.Workspace})
	}
	h.out.Reset()
	h.app.Stdin = strings.NewReader("")
	if code := h.run("equipo"); code != 0 {
		t.Fatalf("segunda %d: %s", code, h.errOut)
	}
	if len(h.hd.runs) != 8 || strings.Contains(h.out.String(), "Puerto") || strings.Count(h.out.String(), "ya corre") != 8 {
		t.Fatalf("relanzó o preguntó: %q\n%s", h.hd.runs, h.out)
	}
}

func TestEquipoConNombresYAgenteQueYaCorre(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "tl")
	if code := h.run("equipo", "tl", "dp", "devbackend", "pp"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if !strings.Contains(h.out.String(), "tl ya corre") || len(h.hd.runs) != 3 {
		t.Fatalf("runs = %q\n%s", h.hd.runs, h.out)
	}
	for i, s := range []string{"dp", "devbackend", "pp"} {
		if !strings.Contains(h.hd.runs[i], "_lanzar "+s+" ") {
			t.Fatalf("run %d = %q", i, h.hd.runs[i])
		}
	}
	if strings.Contains(h.out.String(), "Puerto") {
		t.Fatal("con la config completa no pregunta")
	}
	if code := h.run("equipo", "jefe"); code != 2 || !strings.Contains(h.errOut.String(), "agente desconocido: jefe") {
		t.Fatalf("desconocido %d: %s", code, h.errOut)
	}
}

func TestEquipoConfigurarConValoresActuales(t *testing.T) {
	h := newTeamHarness(t, teamConfig())
	// Descripción, cadena, consulta y puertos con Enter; los perfiles uno por uno:
	// dp a codex con effort high, los demás con Enter.
	answers := "\n\n\n\n\nn\n"
	for _, p := range []string{"pc", "tl", "dp", "devback", "devfront", "cr", "qap", "qa"} {
		if p == "dp" {
			answers += "codex\n\nhigh\n"
		} else {
			answers += "\n\n\n"
		}
	}
	h.app.Stdin = strings.NewReader(answers)
	if code := h.run("equipo", "--configurar", "pc"); code != 0 {
		t.Fatalf("salida %d: %s\n%s", code, h.errOut, h.out)
	}
	out := h.out.String()
	for _, want := range []string{"[Tiene backend/ (Go).]", "Cadena de conexión de la base de datos [Enter conserva la actual]", "Puerto del backend [8090]", "DP:\n  Herramienta (claude o codex) [claude]: "} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q en:\n%s", want, out)
		}
	}
	if strings.Contains(out, "clave-secreta") {
		t.Fatal("la cadena no se imprime")
	}
	got, _ := loadConfig(h.root)
	if got.Base.Cadena != cadenaPrueba || fmt.Sprint(got.Perfiles["dp"]) != fmt.Sprint(ProfileConfig{Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"}) {
		t.Fatalf("config = %+v", got)
	}
	if fmt.Sprint(got.Perfiles["tl"]) != fmt.Sprint(proposed["tl"]) {
		t.Fatalf("tl = %+v", got.Perfiles["tl"])
	}
}

func TestEquipoSinHerdrNiConfig(t *testing.T) {
	h := newTeamHarness(t, teamConfig())
	h.env["HERDR_ENV"] = ""
	if code := h.run("equipo"); code != 1 || !strings.Contains(h.errOut.String(), "dentro de herdr") {
		t.Fatalf("sin herdr %d: %s", code, h.errOut)
	}
	// Los comandos del equipo sin config ni workspace.
	h.root = t.TempDir()
	h.errOut.Reset()
	if code := h.run("avisar", "tl", "hola"); code != 1 || !strings.Contains(h.errOut.String(), "no encuentro .pmt/config.yaml") {
		t.Fatalf("sin config %d: %s", code, h.errOut)
	}
	if err := saveConfig(h.root, teamConfig()); err != nil {
		t.Fatal(err)
	}
	h.errOut.Reset()
	if code := h.run("reiniciar", "tl"); code != 1 || !strings.Contains(h.errOut.String(), "no existe el workspace demo") {
		t.Fatalf("sin workspace %d: %s", code, h.errOut)
	}
}

func TestComandosDesdeSubcarpetaYErroresClaros(t *testing.T) {
	cfg := teamConfig()
	delete(cfg.Perfiles, "qap")
	h := newTeamHarness(t, cfg, "tl", "dp")
	sub := filepath.Join(h.root, "backend", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	h.app.Getwd = func() (string, error) { return sub, nil }
	if code := h.run("avisar", "qap", "hola"); code != 1 || !strings.Contains(h.errOut.String(), "qap no está en .pmt/config.yaml") {
		t.Fatalf("sin config %d: %s", code, h.errOut)
	}
	h.errOut.Reset()
	if code := h.run("reiniciar", "qa"); code != 1 || !strings.Contains(h.errOut.String(), "qa no corre en el workspace demo") {
		t.Fatalf("sin pestaña %d: %s", code, h.errOut)
	}
	// Pestaña dp con el agente cerrado.
	h.hd.stop(h.paneOfAgent("dp"))
	h.hd.agents = h.hd.agents[:1]
	h.errOut.Reset()
	if code := h.run("reiniciar", "dp"); code != 1 || !strings.Contains(h.errOut.String(), "dp no corre en") {
		t.Fatalf("cerrado %d: %s", code, h.errOut)
	}
	if code := h.run("avisar", "tl"); code != 2 {
		t.Fatal("avisar sin mensaje es error de uso")
	}
}

func TestAvisarConfirmado(t *testing.T) {
	cfg := teamConfig()
	cfg.Herdr.Prefijo = "demo-"
	h := newTeamHarness(t, cfg, "tl")
	h.hd.hook = delivers
	// Con el nombre que muestra entorno.md, con el prefijo.
	if code := h.run("avisar", "demo-tl", "Etapa 2 lista:\nrevisa"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	pane := h.paneOfAgent("tl")
	if len(h.hd.prompts) != 1 || h.hd.prompts[0] != pane+": Etapa 2 lista: revisa" {
		t.Fatalf("prompts = %q", h.hd.prompts)
	}
	raw, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "para-usuario.md"))
	if len(raw) != 0 {
		t.Fatalf("para-usuario = %q", raw)
	}
}

func TestAvisarReintentaSinDuplicar(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "devback")
	pane := h.paneOfAgent("devback")
	// El mensaje se queda en el compositor; con Enter se envía.
	h.hd.hook = func(f *fakeHerdr, op, p, arg string) {
		switch {
		case op == "prompt":
			f.screen[p] = "Hola\n❯ " + arg + "\n"
		case op == "keys" && arg == "enter":
			f.screen[p] = "Hola\nlisto el plan\n❯ \n"
		}
	}
	if code := h.run("avisar", "devback", "listo el plan"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if len(h.hd.prompts) != 1 || strings.Join(h.hd.keys, ",") != pane+": enter" {
		t.Fatalf("prompts %q, teclas %q", h.hd.prompts, h.hd.keys)
	}

	// Sin rastro y quieto: reenvía una vez y llega.
	h.hd.prompts, h.hd.keys = nil, nil
	h.hd.screen[pane] = "Hola\n❯ \n"
	n := 0
	h.hd.hook = func(f *fakeHerdr, op, p, arg string) {
		if op == "prompt" {
			if n++; n == 2 {
				delivers(f, op, p, arg)
			}
		}
	}
	if code := h.run("avisar", "devback", "otro aviso"); code != 0 {
		t.Fatalf("reenvío %d: %s", code, h.errOut)
	}
	if len(h.hd.prompts) != 2 {
		t.Fatalf("prompts %q", h.hd.prompts)
	}

	// Ocupado y sin rastro: pudo entrar; no reenvía y lo anota.
	h.hd.prompts = nil
	h.hd.hook = func(f *fakeHerdr, op, p, arg string) {
		if op == "prompt" {
			f.pane(p).Status = "working"
		}
	}
	h.hd.pane(pane).Status = "working"
	if code := h.run("avisar", "devback", "tercero"); code != 1 || !strings.Contains(h.errOut.String(), "ocupado y sin rastro") {
		t.Fatalf("ocupado %d: %s", code, h.errOut)
	}
	if len(h.hd.prompts) != 1 {
		t.Fatalf("prompts %q", h.hd.prompts)
	}
}

func TestAvisarFallaYLoAnota(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "tl")
	pane := h.paneOfAgent("tl")
	// El agente nunca lo recibe.
	if code := h.run("avisar", "tl", "nadie me oye"); code != 1 || !strings.Contains(h.errOut.String(), "aviso sin confirmar a tl (no confirmado tras reintentar)") {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if len(h.hd.prompts) != 2 {
		t.Fatalf("debía reintentar una vez: %q", h.hd.prompts)
	}
	raw, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "para-usuario.md"))
	if !strings.HasSuffix(string(raw), " aviso sin confirmar a tl: nadie me oye\n") || strings.Count(string(raw), "\n") != 1 {
		t.Fatalf("para-usuario = %q", raw)
	}

	// Sin agente en el pane (solo el shell): no envía nada.
	h.hd.prompts = nil
	h.hd.busy[pane] = false
	if code := h.run("avisar", "tl", "hola"); code != 1 || !strings.Contains(h.errOut.String(), "sin agente en "+pane) {
		t.Fatalf("sin agente %d: %s", code, h.errOut)
	}
	if len(h.hd.prompts) != 0 {
		t.Fatal("no debía enviar")
	}

	// Bloqueado en un diálogo: no se envía.
	h.hd.busy[pane] = true
	h.hd.pane(pane).Status = "blocked"
	if code := h.run("avisar", "tl", "hola"); code != 1 || !strings.Contains(h.errOut.String(), "no se pudo enviar") || len(h.hd.prompts) != 0 {
		t.Fatalf("bloqueado %d: %s", code, h.errOut)
	}
}

func TestAvisarCodexConDolar(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "qa")
	pane := h.paneOfAgent("qa")
	// El $ abre el selector; Esc lo cierra y Enter envía.
	h.hd.hook = func(f *fakeHerdr, op, p, arg string) {
		switch {
		case op == "text":
			f.screen[p] = "Hola\n› " + arg + "\n  esc close\n"
		case op == "keys" && arg == "esc":
			f.screen[p] = "Hola\n› costo $5\n"
		case op == "keys" && arg == "enter":
			f.screen[p] = "Hola\ncosto $5\n› \n"
		}
	}
	h.hd.screen[pane] = "Hola\n› \n"
	if code := h.run("avisar", "qa", "costo $5"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if len(h.hd.prompts) != 0 || len(h.hd.texts) != 1 || strings.Join(h.hd.keys, ",") != pane+": esc,"+pane+": enter" {
		t.Fatalf("prompts %q, textos %q, teclas %q", h.hd.prompts, h.hd.texts, h.hd.keys)
	}
}

func TestReiniciarClaudeEsperaAQueTermine(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "cr")
	pane := h.paneOfAgent("cr")
	h.hd.pane(pane).Status = "working"
	checks := 0
	h.app.Sleep = func(_ context.Context, d time.Duration) error {
		if d == 3*time.Second {
			if checks++; checks == 2 {
				h.hd.pane(pane).Status = "idle"
			}
		}
		return nil
	}
	if code := h.run("reiniciar", "cr"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if checks != 2 || strings.Join(h.hd.keys, ",") != pane+": ctrl+u" || strings.Join(h.hd.runs, ",") != pane+": /new" || h.hd.renamed[pane] != "cr" {
		t.Fatalf("esperas %d, teclas %q, runs %q, nombres %v", checks, h.hd.keys, h.hd.runs, h.hd.renamed)
	}
	if !strings.Contains(h.out.String(), "cr reiniciado") {
		t.Fatal(h.out.String())
	}

	// Sigue trabajando 90 s: no lo toca.
	h.hd.runs, h.hd.keys, checks = nil, nil, 0
	h.hd.pane(pane).Status = "working"
	h.app.Sleep = func(context.Context, time.Duration) error { checks++; return nil }
	if code := h.run("reiniciar", "cr"); code != 1 || !strings.Contains(h.errOut.String(), "sigue trabajando tras 90 s") {
		t.Fatalf("90 s %d: %s", code, h.errOut)
	}
	if checks != 30 || len(h.hd.runs) != 0 {
		t.Fatalf("esperas %d, runs %q", checks, h.hd.runs)
	}
	// El título Failed no cuenta como trabajando.
	h.hd.pane(pane).Title = "Failed: error"
	if code := h.run("reiniciar", "cr"); code != 0 {
		t.Fatalf("failed %d: %s", code, h.errOut)
	}
}

func TestReiniciarCodexRelanza(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "devback")
	pane := h.paneOfAgent("devback")
	if code := h.run("reiniciar", "devback"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if strings.Join(h.hd.keys, ",") != pane+": ctrl+u" || len(h.hd.runs) != 2 || h.hd.runs[0] != pane+": /quit" ||
		h.hd.runs[1] != pane+": '/opt/mis cosas/pmt' _lanzar devbackend "+h.root {
		t.Fatalf("teclas %q, runs %q", h.hd.keys, h.hd.runs)
	}
	if h.hd.renamed[pane] != "devback" {
		t.Fatalf("nombres %v", h.hd.renamed)
	}
	if _, err := os.Stat(filepath.Join(h.root, pmtDir, "perfiles", "devbackend.md")); err != nil {
		t.Fatal("el perfil se descarga de nuevo al relanzar")
	}

	// Si no cierra en 10 s, error.
	h.hd.hook = func(f *fakeHerdr, op, p, arg string) {
		if op == "run" && arg == "/quit" {
			f.busy[p] = true
		}
	}
	if code := h.run("reiniciar", "devback"); code != 1 || !strings.Contains(h.errOut.String(), "devback no cerró") {
		t.Fatalf("no cerró %d: %s", code, h.errOut)
	}
}

func TestLevantarTLArrancaVigia(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "tl", "dp")
	h.hd.hook = delivers
	var spawned [][]string
	h.app.Spawn = func(bin string, args []string, dir, logPath string) error {
		spawned = append(spawned, append([]string{bin, dir, logPath}, args...))
		return nil
	}
	if code := h.run("levantar", "tl", "Etapa 1 de Demo: inicia y trabaja"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	pane := h.paneOfAgent("tl")
	if len(spawned) != 1 || strings.Join(spawned[0], " ") != "/opt/mis cosas/pmt "+h.root+" "+filepath.Join(h.root, pmtDir, "vigia.log")+" vigia" {
		t.Fatalf("vigía = %q", spawned)
	}
	if strings.Join(h.hd.runs, ",") != pane+": /new" || len(h.hd.prompts) != 1 || h.hd.prompts[0] != pane+": Etapa 1 de Demo: inicia y trabaja" {
		t.Fatalf("runs %q, prompts %q", h.hd.runs, h.hd.prompts)
	}
	if !strings.Contains(h.out.String(), "tl levantado") {
		t.Fatal(h.out.String())
	}

	// Con el vigía vivo no arranca otro; con otro agente tampoco.
	if err := os.WriteFile(watcherPIDPath(h.root), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := h.run("levantar", "tl", "Etapa 2"); code != 0 {
		t.Fatalf("segunda %d: %s", code, h.errOut)
	}
	os.Remove(watcherPIDPath(h.root))
	if code := h.run("levantar", "dp", "Plan de T1"); code != 0 {
		t.Fatalf("dp %d: %s", code, h.errOut)
	}
	if len(spawned) != 1 {
		t.Fatalf("vigías = %q", spawned)
	}
	if code := h.run("levantar", "dp"); code != 1 || !strings.Contains(h.errOut.String(), "falta el encargo") {
		t.Fatalf("sin encargo %d: %s", code, h.errOut)
	}
}

func TestRelanzarDPGuardaEffort(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "dp")
	pane := h.paneOfAgent("dp")
	if code := h.run("relanzar-dp", "high"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	got, _ := loadConfig(h.root)
	if got.Perfiles["dp"].Effort != "high" || got.Perfiles["dp"].Herramienta != "claude" {
		t.Fatalf("dp = %+v", got.Perfiles["dp"])
	}
	if len(h.hd.runs) != 2 || h.hd.runs[0] != pane+": /exit" || !strings.Contains(h.hd.runs[1], "_lanzar dp ") {
		t.Fatalf("runs %q", h.hd.runs)
	}
	if !strings.Contains(h.out.String(), "dp relanzado en high") {
		t.Fatal(h.out.String())
	}
	// El lanzador toma el effort guardado.
	if code := h.run(launcherCommand, "dp", h.root); code != 0 {
		t.Fatalf("lanzador %d: %s", code, h.errOut)
	}
	if !strings.Contains(strings.Join(h.execs[0], " "), "--effort high") {
		t.Fatalf("exec %q", h.execs[0])
	}
	if code := h.run("relanzar-dp", "low"); code != 2 {
		t.Fatal("solo medium o high")
	}
}

func TestVigiaAvisaAlTLYTerminaConAviso(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "tl", "devback", "pp")
	h.hd.hook = delivers
	tl, dev, pp := h.paneOfAgent("tl"), h.paneOfAgent("devback"), h.paneOfAgent("pp")
	pulses := 0
	h.app.Sleep = func(_ context.Context, d time.Duration) error {
		if d != 15*time.Second {
			return nil
		}
		pulses++
		switch pulses {
		case 1:
			h.hd.pane(dev).Status = "working"
		case 2:
			h.hd.pane(dev).Status = "idle"
			h.hd.pane(pp).Status = "working" // PP no rearma ni cuenta como quieto.
		case 3:
			h.hd.pane(pp).Status = "idle"
		case 20:
			// Alguien anota para el usuario.
			f, _ := os.OpenFile(filepath.Join(h.root, pmtDir, "para-usuario.md"), os.O_APPEND|os.O_WRONLY, 0o644)
			f.WriteString("- etapa 1 cerrada\n")
			f.Close()
		}
		return nil
	}
	if code := h.run("vigia", "--quieto", "45s"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	// Quietos desde el pulso 3: aviso en el 5 (45 s); no repite sin que nadie más trabaje.
	if len(h.hd.prompts) != 1 || !strings.HasPrefix(h.hd.prompts[0], tl+": Vigía: todos llevan 45 s quietos.") {
		t.Fatalf("prompts %q", h.hd.prompts)
	}
	log, _ := os.ReadFile(filepath.Join(h.root, pmtDir, "vigia.log"))
	for _, want := range []string{"vigía en marcha", "TODOS QUIETOS: aviso al TL", "AVISO\n- etapa 1 cerrada"} {
		if !strings.Contains(string(log), want) {
			t.Fatalf("falta %q en la bitácora:\n%s", want, log)
		}
	}
	if pulses != 20 {
		t.Fatalf("pulsos = %d", pulses)
	}
	if _, err := os.Stat(watcherPIDPath(h.root)); !os.IsNotExist(err) {
		t.Fatal("el pid se borra al terminar")
	}
}

func TestVigiaTerminaSiAlguienQuedaBloqueado(t *testing.T) {
	h := newTeamHarness(t, teamConfig(), "tl", "qa")
	h.hd.pane(h.paneOfAgent("qa")).Status = "blocked"
	if code := h.run("vigia"); code != 0 {
		t.Fatalf("salida %d: %s", code, h.errOut)
	}
	if !strings.Contains(h.out.String(), "BLOQUEADO qa:blocked") || len(h.hd.prompts) != 0 {
		t.Fatalf("salida:\n%s", h.out)
	}
	// Con otro vigía vivo no arranca.
	if err := os.WriteFile(watcherPIDPath(h.root), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := h.run("vigia"); code != 1 || !strings.Contains(h.errOut.String(), "el vigía ya corre (pid 1)") {
		t.Fatalf("doble %d: %s", code, h.errOut)
	}
}

func TestObserve(t *testing.T) {
	msg := "Revisa la tarea T-12 del grupo de backend, por favor"
	idle := herdrPane{Status: "idle"}
	if l := observe(idle, "sin compositor", msg); l.count != -1 {
		t.Fatalf("sin compositor: %+v", l)
	}
	if l := observe(idle, "algo\n❯ "+msg+"\n", msg); !l.pending || l.count != 0 {
		t.Fatalf("pendiente: %+v", l)
	}
	if l := observe(idle, msg+"\n❯ \n", msg); l.pending || l.count != 2 {
		t.Fatalf("enviado: %+v", l)
	}
	if l := observe(idle, "x\n›  [Pasted text #1]\n", msg); !l.pending {
		t.Fatalf("pegado: %+v", l)
	}
	if l := observe(herdrPane{Status: "idle", Title: "Working…"}, "❯ \n", msg); !l.busy {
		t.Fatalf("título Working: %+v", l)
	}
	if l := observe(herdrPane{Status: "working", Title: "Failed"}, "❯ \n", msg); l.busy {
		t.Fatalf("Failed no está ocupado: %+v", l)
	}
}
