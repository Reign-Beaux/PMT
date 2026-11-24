package cli

import (
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
	in := newPrompter(ctx, a.Stdin)

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
		found, err := a.adoptNewProject(ctx, in, api, &cfg)
		if err != nil {
			return err
		}
		if found {
			if err := saveConfig(root, cfg); err != nil {
				return err
			}
		}
	}

	if err := writeEnvironment(root, cfg, creds.Username); err != nil {
		return err
	}

	// herdr: workspace, nombre del agente y si PP ya corre.
	ws, err := a.ensureWorkspace(ctx, hd, cfg.Herdr.Workspace, cfg.projectPath(root), "pp")
	if err != nil {
		return err
	}
	if prefix, taken, err := resolvePrefix(ctx, hd, cfg, ws.ID, []string{"pp"}); err != nil {
		return err
	} else if taken != "" {
		cfg.Herdr.Prefijo = prefix
		fmt.Fprintf(a.Stdout, "Ya hay un agente «%s» en otro workspace: este equipo usa el prefijo «%s».\n", taken, prefix)
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

	if err := a.prepareLaunch(ctx, api, cfg, root, creds.Server, "pp"); err != nil {
		return err
	}

	if tab == "" {
		t, p, err := hd.CreateTab(ctx, ws.ID, cfg.projectPath(root), "pp")
		if err != nil {
			return err
		}
		tab, pane = t.ID, p
	}
	if err := a.runLauncher(ctx, hd, pane, root, "pp"); err != nil {
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
func (a *App) askConfig(ctx context.Context, in *prompter, api client, cfg Config, root string) (Config, error) {
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

func (a *App) askText(in *prompter, label, def string) (string, error) {
	fmt.Fprintf(a.Stdout, "%s [%s]: ", label, def)
	v, err := in.readLine()
	if err != nil {
		return "", err
	}
	if v == "" {
		return def, nil
	}
	return v, nil
}

func (a *App) askChoice(in *prompter, label, def string, options []string) (string, error) {
	for range 3 {
		fmt.Fprintf(a.Stdout, "%s [%s]: ", label, def)
		v, err := in.readLine()
		if err != nil {
			return "", err
		}
		if v == "" {
			return def, nil
		}
		for _, o := range options {
			if strings.EqualFold(v, o) {
				return o, nil
			}
		}
		fmt.Fprintf(a.Stdout, "Usa %s.\n", strings.Join(options, " o "))
	}
	return "", fmt.Errorf("respuesta inválida para %s", strings.ToLower(label))
}

func (a *App) askNumber(in *prompter, label string, def, max int) (int, error) {
	for range 3 {
		fmt.Fprintf(a.Stdout, "%s [%d]: ", label, def)
		v, err := in.readLine()
		if err != nil {
			return 0, err
		}
		if v == "" {
			return def, nil
		}
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= max {
			return n, nil
		}
		fmt.Fprintf(a.Stdout, "Escribe un número del 1 al %d.\n", max)
	}
	return 0, errors.New("respuesta inválida")
}

func (a *App) askYes(in *prompter, label string) (bool, error) {
	fmt.Fprintf(a.Stdout, "%s [S/n]: ", label)
	v, err := in.readLine()
	if err != nil {
		return false, err
	}
	v = strings.ToLower(v)
	return v == "" || v == "s" || v == "si" || v == "sí" || v == "y" || v == "yes", nil
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
// confirma, lo guarda en cfg. Solo impide el arranque si la persona corta la
// pregunta (Ctrl+C o Ctrl+D); si algo más falla, avisa.
func (a *App) adoptNewProject(ctx context.Context, in *prompter, api client, cfg *Config) (bool, error) {
	var projects []projectItem
	if err := api.do(ctx, http.MethodGet, "/api/projects", nil, &projects); err != nil {
		fmt.Fprintf(a.Stderr, "Aviso: no pude revisar si PP ya creó el proyecto: %s\n", err)
		return false, nil
	}
	found := newProjectCandidates(projects, *cfg.Proyecto.NuevoDesde)
	var pick *projectItem
	switch len(found) {
	case 0:
		return false, nil
	case 1:
		yes, err := a.askYes(in, fmt.Sprintf("PP creó el proyecto «%s». ¿Es el de esta carpeta?", found[0].Name))
		if err != nil {
			return false, err
		}
		if yes {
			pick = &found[0]
		}
	default:
		fmt.Fprintln(a.Stdout, "Desde que configuraste esta carpeta creaste estos proyectos:")
		for i, p := range found {
			fmt.Fprintf(a.Stdout, "  %d) %s\n", i+1, p.Name)
		}
		fmt.Fprintf(a.Stdout, "  %d) Ninguno todavía\n", len(found)+1)
		n, err := a.askNumber(in, "¿Cuál es el de esta carpeta?", len(found)+1, len(found)+1)
		if errors.Is(err, errInterrupted) || errors.Is(err, errNoInput) {
			return false, err
		}
		if err == nil && n <= len(found) {
			pick = &found[n-1]
		}
	}
	if pick == nil {
		return false, nil
	}
	cfg.Proyecto.ID, cfg.Proyecto.Nombre, cfg.Proyecto.NuevoDesde = pick.ID, pick.Name, nil
	fmt.Fprintf(a.Stdout, "Guardé «%s» como el proyecto de esta carpeta.\n", pick.Name)
	return true, nil
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

// findWorkspace busca el workspace por etiqueta o ID.
func findWorkspace(ctx context.Context, hd herdr, label string) (herdrWorkspace, bool, error) {
	list, err := hd.Workspaces(ctx)
	if err != nil {
		return herdrWorkspace{}, false, err
	}
	for _, w := range list {
		if w.Label == label || w.ID == label {
			return w, true, nil
		}
	}
	return herdrWorkspace{}, false, nil
}

// prepareLaunch deja listo lo que el lanzador lee de .pmt/ para un perfil: el
// perfil descargado y, con Claude, mcp-<perfil>.json. Valida ya los MCP extra,
// para fallar aquí y no en el pane.
func (a *App) prepareLaunch(ctx context.Context, api client, cfg Config, root, server, session string) error {
	p, ok := cfg.profile(session)
	if !ok {
		return fmt.Errorf("el perfil %s no está configurado en .pmt/config.yaml", session)
	}
	if err := a.downloadProfile(ctx, api, root, session); err != nil {
		return err
	}
	if p.Herramienta == "claude" {
		raw, err := mcpConfigJSON(cfg, server, session)
		if err != nil {
			return err
		}
		if err := writeFileAtomic(filepath.Join(root, pmtDir, "mcp-"+session+".json"), raw, 0o644); err != nil {
			return err
		}
	}
	extras, _, err := extraMCP(cfg, session, a.Home)
	if err != nil {
		return err
	}
	if p.Herramienta == "codex" {
		// El config.toml roto se reporta aquí y no en el pane; el aviso de
		// entradas stdio va una vez por archivo.
		stdio, file, err := codexStdioServers(cfg.codexHome(a.Home))
		if err != nil {
			return err
		}
		for _, name := range append([]string{"pmt"}, extras...) {
			key := file + "\x00" + name
			if stdio[name] && !a.codexNoticed[key] {
				if a.codexNoticed == nil {
					a.codexNoticed = map[string]bool{}
				}
				a.codexNoticed[key] = true
				fmt.Fprintln(a.Stdout, codexStdioWarning(file, name))
			}
		}
	}
	return nil
}

// runLauncher escribe en el pane «pmt _lanzar <perfil> <raíz>». El comando no
// lleva el token: el lanzador lo lee de las credenciales y lo pone en el
// entorno del agente.
func (a *App) runLauncher(ctx context.Context, hd herdr, pane, root, session string) error {
	exe, err := a.Executable()
	if err != nil {
		return err
	}
	return hd.Run(ctx, pane, shellQuote(exe)+" "+launcherCommand+" "+session+" "+shellQuote(root))
}

// ensureWorkspace busca el workspace por etiqueta o ID y lo crea si no existe.
// Al crearlo, su primera pestaña queda como la del agente first.
func (a *App) ensureWorkspace(ctx context.Context, hd herdr, label, cwd, first string) (herdrWorkspace, error) {
	if w, ok, err := findWorkspace(ctx, hd, label); err != nil || ok {
		return w, err
	}
	w, tab, _, err := hd.CreateWorkspace(ctx, cwd, label)
	if err != nil {
		return herdrWorkspace{}, err
	}
	if err := hd.RenameTab(ctx, tab.ID, first); err != nil {
		return herdrWorkspace{}, err
	}
	fmt.Fprintf(a.Stdout, "Creé el workspace %s en herdr.\n", label)
	return w, nil
}

var prefixClean = regexp.MustCompile(`[^a-z0-9]+`)

// resolvePrefix decide el prefijo de los nombres: si otro workspace ya tiene
// un agente con el nombre de alguno de agents y no hay prefijo, usa el del
// workspace. taken es el nombre que chocó, si el prefijo cambió.
func resolvePrefix(ctx context.Context, hd herdr, cfg Config, workspace string, agents []string) (prefix, taken string, err error) {
	list, err := hd.Agents(ctx)
	if err != nil {
		return "", "", err
	}
	for _, agent := range agents {
		name := cfg.agentName(agent)
		for _, ag := range list {
			if ag.Name != name || ag.Workspace == workspace {
				continue
			}
			if cfg.Herdr.Prefijo != "" {
				return cfg.Herdr.Prefijo, "", fmt.Errorf("ya hay un agente «%s» en otro workspace: cambia herdr.prefijo en .pmt/config.yaml", name)
			}
			p := strings.Trim(prefixClean.ReplaceAllString(strings.ToLower(cfg.Herdr.Workspace), "-"), "-")
			if p == "" {
				p = "equipo"
			}
			return p + "-", name, nil
		}
	}
	return cfg.Herdr.Prefijo, "", nil
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
