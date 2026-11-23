package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Herramientas de los agentes y sus modelos por omisión.
var defaultModel = map[string]string{"claude": "opus", "codex": "gpt-6.1-terra"}

// projectItem es un proyecto de GET /api/projects.
type projectItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// pp levanta a PP en herdr para el proyecto de la carpeta actual.
func (a *App) pp(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("pp", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	reconfigure := fs.Bool("configurar", false, "vuelve a preguntar la configuración, con los valores actuales por omisión")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{"pp solo recibe --configurar"}
	}

	if a.Getenv("HERDR_ENV") != "1" {
		return errors.New("pmt pp corre dentro de herdr: abre herdr con el comando «herdr» y corre «pmt pp» en uno de sus panes, en la carpeta del proyecto")
	}
	creds, err := loadCredentials(a.Home)
	if err != nil {
		return err
	}
	hd, err := a.Herdr()
	if err != nil {
		return err
	}
	cwd, err := a.Getwd()
	if err != nil {
		return err
	}
	root := findRoot(cwd)
	api := a.client(creds.Server, creds.Token)
	in := bufio.NewReader(a.Stdin)

	cfg, err := loadConfig(root)
	switch {
	case errors.Is(err, errNoConfig):
		fmt.Fprintf(a.Stdout, "Configurando pmt para %s (se guarda en .pmt/config.yaml).\n", root)
		cfg = defaultConfig(root)
		if cfg, err = a.askConfig(ctx, in, api, cfg, root); err != nil {
			return err
		}
	case err != nil:
		return err
	case *reconfigure:
		if cfg, err = a.askConfig(ctx, in, api, cfg, root); err != nil {
			return err
		}
	}
	if cfg.Herdr.Workspace == "" {
		cfg.Herdr.Workspace = filepath.Base(root)
	}
	pp, ok := cfg.profile("pp")
	if !ok {
		return errors.New("falta perfiles.pp en .pmt/config.yaml: corre «pmt pp --configurar»")
	}
	if _, err := a.LookPath(pp.Herramienta); err != nil {
		return fmt.Errorf("no encuentro %s en el PATH: instálalo o elige otra herramienta con «pmt pp --configurar»", pp.Herramienta)
	}
	if err := saveConfig(root, cfg); err != nil {
		return err
	}
	if isGitRepo(root) {
		changed, err := ensureGitignore(root)
		if err != nil {
			return err
		}
		if changed {
			fmt.Fprintln(a.Stdout, "Agregué .pmt/ al .gitignore del proyecto.")
		}
	}

	if cfg.Proyecto.ID == "" && cfg.Proyecto.NuevoDesde != nil {
		if found := a.adoptNewProject(ctx, in, api, &cfg); found {
			if err := saveConfig(root, cfg); err != nil {
				return err
			}
		}
	}

	if err := writeEnvironment(root, cfg, creds.Username); err != nil {
		return err
	}

	// herdr: workspace, nombre del agente y si PP ya corre.
	ws, err := a.ensureWorkspace(ctx, hd, cfg.Herdr.Workspace, cfg.projectPath(root))
	if err != nil {
		return err
	}
	if prefix, changed, err := resolvePrefix(ctx, hd, cfg, ws.ID); err != nil {
		return err
	} else if changed {
		cfg.Herdr.Prefijo = prefix
		fmt.Fprintf(a.Stdout, "Ya hay un agente «pp» en otro workspace: este equipo usa el prefijo «%s».\n", prefix)
		if err := saveConfig(root, cfg); err != nil {
			return err
		}
		if err := writeEnvironment(root, cfg, creds.Username); err != nil {
			return err
		}
	}
	name := cfg.agentName("pp")
	tab, pane, running, err := a.findAgentPane(ctx, hd, ws.ID, "pp", name)
	if err != nil {
		return err
	}
	if running {
		fmt.Fprintf(a.Stdout, "PP ya corre en el workspace %s (pestaña pp): no lo relanzo.\n", ws.Label)
		return hd.FocusTab(ctx, tab)
	}

	// Perfil y MCP.
	if err := a.downloadProfile(ctx, api, root, "pp"); err != nil {
		return err
	}
	if pp.Herramienta == "claude" {
		raw, err := mcpConfigJSON(cfg, creds.Server, "pp")
		if err != nil {
			return err
		}
		if err := writeFileAtomic(filepath.Join(root, pmtDir, "mcp-pp.json"), raw, 0o644); err != nil {
			return err
		}
	}
	// Se valida ya lo que el lanzador necesita, para fallar aquí y no en el pane.
	if _, _, err := extraMCP(cfg, "pp", a.Home); err != nil {
		return err
	}

	if tab == "" {
		t, p, err := hd.CreateTab(ctx, ws.ID, cfg.projectPath(root), "pp")
		if err != nil {
			return err
		}
		tab, pane = t.ID, p
	}
	exe, err := a.Executable()
	if err != nil {
		return err
	}
	// El comando que se escribe en el pane no lleva el token: el lanzador lo lee
	// de las credenciales y lo pone en el entorno del agente.
	if err := hd.Run(ctx, pane, shellQuote(exe)+" "+launcherCommand+" pp "+shellQuote(root)); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Lanzando PP con %s (%s, effort %s) en el workspace %s…\n", pp.Herramienta, pp.Modelo, pp.Effort, ws.Label)
	if a.waitAgent(ctx, hd, ws.ID, pane) {
		if err := hd.RenameAgent(ctx, pane, name); err != nil {
			fmt.Fprintf(a.Stderr, "Aviso: no pude nombrar al agente %s: %s\n", name, err)
		}
		fmt.Fprintf(a.Stdout, "PP corre como «%s». Descríbele el proyecto en su pestaña.\n", name)
	} else {
		fmt.Fprintf(a.Stderr, "Aviso: herdr aún no detecta a PP en su pestaña; revísala. Si arranca, nómbralo con «herdr agent rename %s %s».\n", pane, name)
	}
	return hd.FocusTab(ctx, tab)
}

// defaultConfig es la configuración inicial de un proyecto.
func defaultConfig(root string) Config {
	return Config{
		Version: 1,
		Herdr:   HerdrConfig{Workspace: filepath.Base(root)},
		Perfiles: map[string]ProfileConfig{
			"pp": {Herramienta: "claude", Modelo: defaultModel["claude"], Effort: "medium"},
		},
	}
}

// askConfig pregunta proyecto, herramienta, modelo y effort de PP, con los
// valores de cfg por omisión.
func (a *App) askConfig(ctx context.Context, in *bufio.Reader, api client, cfg Config, root string) (Config, error) {
	var projects []projectItem
	if err := api.do(ctx, http.MethodGet, "/api/projects", nil, &projects); err != nil {
		if isStatus(err, http.StatusUnauthorized) {
			return cfg, errors.New("tu sesión ya no es válida; corre «pmt login»")
		}
		return cfg, fmt.Errorf("no pude leer tus proyectos: %w", err)
	}
	workable := projects[:0]
	for _, p := range projects {
		if p.Role == "owner" || p.Role == "editor" {
			workable = append(workable, p)
		}
	}

	// Proyecto: el actual, o uno que se llame como la carpeta, o nuevo.
	def := len(workable) + 1
	for i, p := range workable {
		if (cfg.Proyecto.ID != "" && p.ID == cfg.Proyecto.ID) ||
			(cfg.Proyecto.ID == "" && cfg.Proyecto.NuevoDesde == nil && strings.EqualFold(p.Name, filepath.Base(root))) {
			def = i + 1
		}
	}
	fmt.Fprintln(a.Stdout, "\nProyecto de PMT:")
	for i, p := range workable {
		fmt.Fprintf(a.Stdout, "  %d) %s (%s)\n", i+1, p.Name, p.Role)
	}
	fmt.Fprintf(a.Stdout, "  %d) Nuevo: lo crea PP\n", len(workable)+1)
	n, err := a.askNumber(in, "Elige", def, len(workable)+1)
	if err != nil {
		return cfg, err
	}
	if n <= len(workable) {
		cfg.Proyecto.ID, cfg.Proyecto.Nombre, cfg.Proyecto.NuevoDesde = workable[n-1].ID, workable[n-1].Name, nil
	} else if cfg.Proyecto.ID != "" || cfg.Proyecto.NuevoDesde == nil {
		now := a.Now().UTC().Truncate(time.Second)
		cfg.Proyecto.ID, cfg.Proyecto.Nombre, cfg.Proyecto.NuevoDesde = "", "", &now
	}

	pp, _ := cfg.profile("pp")
	tool, err := a.askChoice(in, "Herramienta de PP (claude o codex)", orDefault(pp.Herramienta, "claude"), []string{"claude", "codex"})
	if err != nil {
		return cfg, err
	}
	modelDef := defaultModel[tool]
	if tool == pp.Herramienta && pp.Modelo != "" {
		modelDef = pp.Modelo
	}
	model, err := a.askText(in, "Modelo", modelDef)
	if err != nil {
		return cfg, err
	}
	effort, err := a.askChoice(in, "Effort (medium o high)", orDefault(pp.Effort, "medium"), []string{"medium", "high"})
	if err != nil {
		return cfg, err
	}
	pp.Herramienta, pp.Modelo, pp.Effort = tool, model, effort
	if cfg.Perfiles == nil {
		cfg.Perfiles = map[string]ProfileConfig{}
	}
	cfg.Perfiles["pp"] = pp
	fmt.Fprintln(a.Stdout)
	return cfg, nil
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// readLine lee una respuesta. Sin más entrada devuelve vacío: el valor por omisión.
func readLine(in *bufio.Reader) (string, bool) {
	line, err := in.ReadString('\n')
	return strings.TrimSpace(line), err == nil || line != ""
}

func (a *App) askText(in *bufio.Reader, label, def string) (string, error) {
	fmt.Fprintf(a.Stdout, "%s [%s]: ", label, def)
	v, _ := readLine(in)
	if v == "" {
		return def, nil
	}
	return v, nil
}

func (a *App) askChoice(in *bufio.Reader, label, def string, options []string) (string, error) {
	for range 3 {
		fmt.Fprintf(a.Stdout, "%s [%s]: ", label, def)
		v, more := readLine(in)
		if v == "" {
			return def, nil
		}
		for _, o := range options {
			if strings.EqualFold(v, o) {
				return o, nil
			}
		}
		fmt.Fprintf(a.Stdout, "Usa %s.\n", strings.Join(options, " o "))
		if !more {
			break
		}
	}
	return "", fmt.Errorf("respuesta inválida para %s", strings.ToLower(label))
}

func (a *App) askNumber(in *bufio.Reader, label string, def, max int) (int, error) {
	for range 3 {
		fmt.Fprintf(a.Stdout, "%s [%d]: ", label, def)
		v, more := readLine(in)
		if v == "" {
			return def, nil
		}
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= max {
			return n, nil
		}
		fmt.Fprintf(a.Stdout, "Escribe un número del 1 al %d.\n", max)
		if !more {
			break
		}
	}
	return 0, errors.New("respuesta inválida")
}

func (a *App) askYes(in *bufio.Reader, label string) bool {
	fmt.Fprintf(a.Stdout, "%s [S/n]: ", label)
	v, _ := readLine(in)
	v = strings.ToLower(v)
	return v == "" || v == "s" || v == "si" || v == "sí" || v == "y" || v == "yes"
}

// newProjectCandidates son los proyectos de los que el usuario es owner,
// creados después de que eligió «nuevo».
func newProjectCandidates(projects []projectItem, since time.Time) []projectItem {
	var out []projectItem
	for _, p := range projects {
		if p.Role == "owner" && !p.CreatedAt.Before(since) {
			out = append(out, p)
		}
	}
	return out
}

// adoptNewProject busca el proyecto que PP debió crear y, si la persona lo
// confirma, lo guarda en cfg. Nunca impide el arranque: si algo falla, avisa.
func (a *App) adoptNewProject(ctx context.Context, in *bufio.Reader, api client, cfg *Config) bool {
	var projects []projectItem
	if err := api.do(ctx, http.MethodGet, "/api/projects", nil, &projects); err != nil {
		fmt.Fprintf(a.Stderr, "Aviso: no pude revisar si PP ya creó el proyecto: %s\n", err)
		return false
	}
	found := newProjectCandidates(projects, *cfg.Proyecto.NuevoDesde)
	var pick *projectItem
	switch len(found) {
	case 0:
		return false
	case 1:
		if a.askYes(in, fmt.Sprintf("PP creó el proyecto «%s». ¿Es el de esta carpeta?", found[0].Name)) {
			pick = &found[0]
		}
	default:
		fmt.Fprintln(a.Stdout, "Desde que configuraste esta carpeta creaste estos proyectos:")
		for i, p := range found {
			fmt.Fprintf(a.Stdout, "  %d) %s\n", i+1, p.Name)
		}
		fmt.Fprintf(a.Stdout, "  %d) Ninguno todavía\n", len(found)+1)
		n, err := a.askNumber(in, "¿Cuál es el de esta carpeta?", len(found)+1, len(found)+1)
		if err == nil && n <= len(found) {
			pick = &found[n-1]
		}
	}
	if pick == nil {
		return false
	}
	cfg.Proyecto.ID, cfg.Proyecto.Nombre, cfg.Proyecto.NuevoDesde = pick.ID, pick.Name, nil
	fmt.Fprintf(a.Stdout, "Guardé «%s» como el proyecto de esta carpeta.\n", pick.Name)
	return true
}

// writeEnvironment regenera .pmt/entorno.md y crea .pmt/para-usuario.md si falta.
func writeEnvironment(root string, cfg Config, username string) error {
	dir := filepath.Join(root, pmtDir)
	if err := writeFileAtomic(filepath.Join(dir, "entorno.md"), []byte(renderEntorno(cfg, root, username)), 0o644); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "para-usuario.md"), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

// downloadProfile guarda el perfil servido en .pmt/perfiles/<perfil>.md. Si el
// servidor no responde y hay copia, usa la copia y avisa.
func (a *App) downloadProfile(ctx context.Context, api client, root, profile string) error {
	path := filepath.Join(root, pmtDir, "perfiles", profile+".md")
	text, err := api.getText(ctx, "/api/profiles/"+profile)
	if err == nil && strings.TrimSpace(text) != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return writeFileAtomic(path, []byte(text), 0o644)
	}
	if isStatus(err, http.StatusUnauthorized) {
		return errors.New("tu sesión ya no es válida; corre «pmt login»")
	}
	if err == nil {
		err = errors.New("el servidor devolvió un perfil vacío")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		fmt.Fprintf(a.Stderr, "Aviso: no pude descargar el perfil %s (%s); uso la copia de .pmt/perfiles/%s.md.\n", profile, err, profile)
		return nil
	}
	return fmt.Errorf("no pude descargar el perfil %s y no hay copia local: %w", profile, err)
}

// ensureWorkspace busca el workspace por etiqueta o ID y lo crea si no existe.
// Al crearlo, su primera pestaña queda como la de PP.
func (a *App) ensureWorkspace(ctx context.Context, hd herdr, label, cwd string) (herdrWorkspace, error) {
	list, err := hd.Workspaces(ctx)
	if err != nil {
		return herdrWorkspace{}, err
	}
	for _, w := range list {
		if w.Label == label || w.ID == label {
			return w, nil
		}
	}
	w, tab, _, err := hd.CreateWorkspace(ctx, cwd, label)
	if err != nil {
		return herdrWorkspace{}, err
	}
	if err := hd.RenameTab(ctx, tab.ID, "pp"); err != nil {
		return herdrWorkspace{}, err
	}
	fmt.Fprintf(a.Stdout, "Creé el workspace %s en herdr.\n", label)
	return w, nil
}

var prefixClean = regexp.MustCompile(`[^a-z0-9]+`)

// resolvePrefix decide el prefijo de los nombres: si otro workspace ya tiene
// un agente con el nombre de PP y no hay prefijo, usa el del workspace.
func resolvePrefix(ctx context.Context, hd herdr, cfg Config, workspace string) (string, bool, error) {
	agents, err := hd.Agents(ctx)
	if err != nil {
		return "", false, err
	}
	name := cfg.agentName("pp")
	for _, ag := range agents {
		if ag.Name == name && ag.Workspace != workspace {
			if cfg.Herdr.Prefijo != "" {
				return cfg.Herdr.Prefijo, false, fmt.Errorf("ya hay un agente «%s» en otro workspace: cambia herdr.prefijo en .pmt/config.yaml", name)
			}
			p := strings.Trim(prefixClean.ReplaceAllString(strings.ToLower(cfg.Herdr.Workspace), "-"), "-")
			if p == "" {
				p = "equipo"
			}
			return p + "-", true, nil
		}
	}
	return cfg.Herdr.Prefijo, false, nil
}

// findAgentPane busca la pestaña de un agente en el workspace: el pane del
// agente con ese nombre o, si no, el de la pestaña con su etiqueta. running
// indica que ahí ya corre algo además del shell.
func (a *App) findAgentPane(ctx context.Context, hd herdr, workspace, label, name string) (tab, pane string, running bool, err error) {
	agents, err := hd.Agents(ctx)
	if err != nil {
		return "", "", false, err
	}
	panes, err := hd.Panes(ctx, workspace)
	if err != nil {
		return "", "", false, err
	}
	for _, ag := range agents {
		if ag.Name == name && ag.Workspace == workspace {
			for _, p := range panes {
				if p.ID == ag.Pane {
					return p.Tab, p.ID, true, nil
				}
			}
		}
	}
	tabs, err := hd.Tabs(ctx, workspace)
	if err != nil {
		return "", "", false, err
	}
	for _, t := range tabs {
		if t.Label != label {
			continue
		}
		for _, p := range panes {
			if p.Tab != t.ID {
				continue
			}
			if p.Agent != "" {
				return t.ID, p.ID, true, nil
			}
			// Un shell recién abierto tarda en quedar libre: se espera hasta 10 s.
			for range 10 {
				busy, err := hd.Busy(ctx, p.ID)
				if err != nil {
					return "", "", false, err
				}
				if !busy {
					return t.ID, p.ID, false, nil
				}
				if err := a.Sleep(ctx, time.Second); err != nil {
					return "", "", false, err
				}
			}
			return t.ID, p.ID, true, nil
		}
	}
	return "", "", false, nil
}

// waitAgent espera a que herdr detecte un agente en el pane, hasta un minuto.
func (a *App) waitAgent(ctx context.Context, hd herdr, workspace, pane string) bool {
	for range 60 {
		if err := a.Sleep(ctx, time.Second); err != nil {
			return false
		}
		panes, err := hd.Panes(ctx, workspace)
		if err != nil {
			continue
		}
		for _, p := range panes {
			if p.ID == pane && p.Agent != "" {
				return true
			}
		}
	}
	return false
}
