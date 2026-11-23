package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// teamDefault son los perfiles que levanta pmt equipo sin nombres: todos menos PP.
var teamDefault = []string{"pc", "tl", "dp", "devback", "devfront", "cr", "qap", "qa"}

// proposed es la herramienta, modelo y effort que pmt equipo propone por perfil.
var proposed = map[string]ProfileConfig{
	"pc":          {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"},
	"tl":          {Herramienta: "claude", Modelo: "opus", Effort: "medium"},
	"dp":          {Herramienta: "claude", Modelo: "opus", Effort: "medium"},
	"devbackend":  {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"},
	"devfrontend": {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"},
	"cr":          {Herramienta: "claude", Modelo: "opus", Effort: "medium"},
	"qap":         {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"},
	"qa":          {Herramienta: "codex", Modelo: "gpt-6.1-terra", Effort: "high"},
}

// lookupProfile busca un perfil por su nombre en herdr (devback) o de sesión (devbackend).
func lookupProfile(name string) (profileInfo, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, p := range allProfiles {
		if p.Agent == name || p.Session == name {
			return p, true
		}
	}
	return profileInfo{}, false
}

// lookupAgent es lookupProfile que acepta además el nombre en herdr con el
// prefijo del equipo (p4c-tl), que es el que muestra entorno.md.
func (c Config) lookupAgent(name string) (profileInfo, bool) {
	if p, ok := lookupProfile(name); ok {
		return p, true
	}
	if c.Herdr.Prefijo != "" && strings.HasPrefix(name, c.Herdr.Prefijo) {
		return lookupProfile(strings.TrimPrefix(name, c.Herdr.Prefijo))
	}
	return profileInfo{}, false
}

// projectDetail es lo que pmt equipo lee de GET /api/projects/{id}.
type projectDetail struct {
	Description string `json:"description"`
}

// equipo levanta los perfiles que falten en el workspace del proyecto.
func (a *App) equipo(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("equipo", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	reconfigure := fs.Bool("configurar", false, "vuelve a preguntar descripción, base, puertos y perfiles, con los valores actuales por omisión")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{err.Error()}
	}
	var wanted []profileInfo
	names := fs.Args()
	if len(names) == 0 {
		names = teamDefault
	}
	for _, n := range names {
		p, ok := lookupProfile(n)
		if !ok {
			return usageError{fmt.Sprintf("agente desconocido: %s (usa pp, pc, tl, dp, devback, devfront, cr, qap o qa)", n)}
		}
		dup := false
		for _, w := range wanted {
			dup = dup || w.Session == p.Session
		}
		if !dup {
			wanted = append(wanted, p)
		}
	}

	if a.Getenv("HERDR_ENV") != "1" {
		return errors.New("pmt equipo corre dentro de herdr: abre herdr con el comando «herdr» y corre «pmt equipo» en uno de sus panes, en la carpeta del proyecto")
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
		// Sin pmt pp antes: se pregunta también lo de PP.
		fmt.Fprintf(a.Stdout, "Configurando pmt para %s (se guarda en .pmt/config.yaml).\n", root)
		if cfg, err = a.askConfig(ctx, in, api, defaultConfig(root), root); err != nil {
			return err
		}
	case err != nil:
		return err
	}
	if *reconfigure || teamIncomplete(cfg) {
		if !*reconfigure {
			fmt.Fprintln(a.Stdout, "Faltan datos del equipo en .pmt/config.yaml; Enter acepta el valor entre corchetes.")
		}
		if cfg, err = a.askTeam(ctx, in, api, cfg, *reconfigure); err != nil {
			return err
		}
	}
	if cfg.Herdr.Workspace == "" {
		cfg.Herdr.Workspace = filepath.Base(root)
	}
	for _, p := range wanted {
		pc, ok := cfg.profile(p.Session)
		if !ok {
			return fmt.Errorf("falta perfiles.%s en .pmt/config.yaml: corre «pmt equipo --configurar»", p.Session)
		}
		if _, err := a.LookPath(pc.Herramienta); err != nil {
			return fmt.Errorf("no encuentro %s en el PATH (lo usa %s): instálalo o cambia la herramienta con «pmt equipo --configurar»", pc.Herramienta, p.Agent)
		}
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
		if a.adoptNewProject(ctx, in, api, &cfg) {
			if err := saveConfig(root, cfg); err != nil {
				return err
			}
		}
	}
	if err := writeEnvironment(root, cfg, creds.Username); err != nil {
		return err
	}

	ws, err := a.ensureWorkspace(ctx, hd, cfg.Herdr.Workspace, cfg.projectPath(root), wanted[0].Agent)
	if err != nil {
		return err
	}
	agentNames := make([]string, len(wanted))
	for i, p := range wanted {
		agentNames[i] = p.Agent
	}
	if prefix, taken, err := resolvePrefix(ctx, hd, cfg, ws.ID, agentNames); err != nil {
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

	type started struct {
		info profileInfo
		pane string
	}
	var launched []started
	for _, p := range wanted {
		name := cfg.agentName(p.Agent)
		tab, pane, running, err := a.findAgentPane(ctx, hd, ws.ID, p.Agent, name)
		if err != nil {
			return err
		}
		if running {
			fmt.Fprintf(a.Stdout, "%s ya corre en %s: no lo toco.\n", p.Agent, pane)
			continue
		}
		if err := a.prepareLaunch(ctx, api, cfg, root, creds.Server, p.Session); err != nil {
			return err
		}
		if tab == "" {
			if _, pane, err = hd.CreateTab(ctx, ws.ID, cfg.projectPath(root), p.Agent); err != nil {
				return err
			}
		}
		if err := a.runLauncher(ctx, hd, pane, root, p.Session); err != nil {
			return err
		}
		pc, _ := cfg.profile(p.Session)
		fmt.Fprintf(a.Stdout, "%s lanzado en %s con %s (%s, effort %s)\n", p.Agent, pane, pc.Herramienta, pc.Modelo, pc.Effort)
		launched = append(launched, started{p, pane})
	}
	for _, s := range launched {
		name := cfg.agentName(s.info.Agent)
		if !a.waitAgent(ctx, hd, ws.ID, s.pane) {
			fmt.Fprintf(a.Stderr, "Aviso: herdr aún no detecta a %s en su pestaña; revísala. Si arranca, nómbralo con «herdr agent rename %s %s».\n", s.info.Agent, s.pane, name)
			continue
		}
		if err := hd.RenameAgent(ctx, s.pane, name); err != nil {
			fmt.Fprintf(a.Stderr, "Aviso: no pude nombrar al agente %s: %s\n", name, err)
		}
	}
	fmt.Fprintf(a.Stdout, "Listo: workspace %s, entorno en .pmt/entorno.md.\n", ws.Label)
	return nil
}

// teamIncomplete indica si faltan datos que pregunta pmt equipo.
func teamIncomplete(c Config) bool {
	if strings.TrimSpace(c.Proyecto.Descripcion) == "" || c.Base.Cadena == "" || c.Base.Consulta == "" ||
		c.Puertos.Backend <= 0 || c.Puertos.Frontend <= 0 {
		return true
	}
	for s := range proposed {
		if _, ok := c.profile(s); !ok {
			return true
		}
	}
	return false
}

// askTeam pregunta descripción, base, puertos y los ocho perfiles del equipo.
// Con all pregunta todo con los valores actuales; si no, solo lo que falta.
func (a *App) askTeam(ctx context.Context, in *bufio.Reader, api client, cfg Config, all bool) (Config, error) {
	var err error
	if all || strings.TrimSpace(cfg.Proyecto.Descripcion) == "" {
		def := cfg.Proyecto.Descripcion
		if def == "" && cfg.Proyecto.ID != "" {
			var p projectDetail
			if err := api.do(ctx, http.MethodGet, "/api/projects/"+cfg.Proyecto.ID, nil, &p); err == nil {
				def = strings.TrimSpace(p.Description)
			}
		}
		if cfg.Proyecto.Descripcion, err = a.askRequired(in, "Descripción del proyecto (qué tiene: backend/, frontend/, tecnologías)", def, false, nil); err != nil {
			return cfg, err
		}
	}
	if all || cfg.Base.Cadena == "" {
		// La cadena lleva la contraseña: no se muestra, se ofrece conservarla.
		if cfg.Base.Cadena, err = a.askRequired(in, "Cadena de conexión de la base de datos", cfg.Base.Cadena, true, nil); err != nil {
			return cfg, err
		}
	}
	if all || cfg.Base.Consulta == "" {
		hasPlaceholder := func(v string) error {
			if !strings.Contains(v, "<consulta>") {
				return errors.New("debe llevar <consulta> donde va la consulta")
			}
			return nil
		}
		if cfg.Base.Consulta, err = a.askRequired(in, "Comando para consultar la base, con <consulta> donde va la consulta (ej. docker exec postgresql psql -U usuario -d base -c \"<consulta>\")", cfg.Base.Consulta, false, hasPlaceholder); err != nil {
			return cfg, err
		}
	}
	if all || cfg.Puertos.Backend <= 0 {
		if cfg.Puertos.Backend, err = a.askPort(in, "Puerto del backend", cfg.Puertos.Backend, 8090); err != nil {
			return cfg, err
		}
	}
	if all || cfg.Puertos.Frontend <= 0 {
		if cfg.Puertos.Frontend, err = a.askPort(in, "Puerto del frontend", cfg.Puertos.Frontend, 5173); err != nil {
			return cfg, err
		}
	}

	// Perfiles: los que faltan (o todos con --configurar), con lo actual o lo propuesto.
	var ask []profileInfo
	plan := map[string]ProfileConfig{}
	for _, p := range allProfiles {
		def, ok := proposed[p.Session]
		if !ok {
			continue
		}
		cur, has := cfg.profile(p.Session)
		if has && !all {
			continue
		}
		if has {
			def = cur
		}
		ask = append(ask, p)
		plan[p.Session] = def
	}
	if len(ask) == 0 {
		fmt.Fprintln(a.Stdout)
		return cfg, nil
	}
	fmt.Fprintln(a.Stdout, "\nPerfiles:")
	for _, p := range ask {
		d := plan[p.Session]
		fmt.Fprintf(a.Stdout, "  %-9s %-7s %-15s %s\n", p.Agent, d.Herramienta, d.Modelo, d.Effort)
	}
	if cfg.Perfiles == nil {
		cfg.Perfiles = map[string]ProfileConfig{}
	}
	acceptAll := a.askYes(in, "¿Los dejo así?")
	for _, p := range ask {
		d := plan[p.Session]
		if !acceptAll {
			fmt.Fprintf(a.Stdout, "%s:\n", p.Label)
			tool, err := a.askChoice(in, "  Herramienta (claude o codex)", d.Herramienta, []string{"claude", "codex"})
			if err != nil {
				return cfg, err
			}
			modelDef := d.Modelo
			if tool != d.Herramienta {
				modelDef = defaultModel[tool]
			}
			model, err := a.askText(in, "  Modelo", modelDef)
			if err != nil {
				return cfg, err
			}
			effort, err := a.askChoice(in, "  Effort (medium o high)", orDefault(d.Effort, "medium"), []string{"medium", "high"})
			if err != nil {
				return cfg, err
			}
			d.Herramienta, d.Modelo, d.Effort = tool, model, effort
		}
		// Los MCP extra que ya tuviera el perfil se conservan.
		d.MCP = cfg.Perfiles[p.Session].MCP
		cfg.Perfiles[p.Session] = d
	}
	fmt.Fprintln(a.Stdout)
	return cfg, nil
}

// askRequired pregunta un texto obligatorio. Con secret no muestra el valor
// actual: Enter lo conserva. check valida la respuesta.
func (a *App) askRequired(in *bufio.Reader, label, def string, secret bool, check func(string) error) (string, error) {
	for range 3 {
		switch {
		case def == "":
			fmt.Fprintf(a.Stdout, "%s: ", label)
		case secret:
			fmt.Fprintf(a.Stdout, "%s [Enter conserva la actual]: ", label)
		default:
			fmt.Fprintf(a.Stdout, "%s [%s]: ", label, def)
		}
		v, more := readLine(in)
		if v == "" {
			v = def
		}
		if v == "" {
			fmt.Fprintln(a.Stdout, "Hace falta un valor.")
		} else if check != nil {
			if err := check(v); err != nil {
				fmt.Fprintf(a.Stdout, "No sirve: %s.\n", err)
				v = ""
			}
		}
		if v != "" {
			return v, nil
		}
		if !more {
			break
		}
	}
	return "", fmt.Errorf("falta %s", strings.ToLower(strings.SplitN(label, " (", 2)[0]))
}

// askPort pregunta un puerto, con el actual o fallback por omisión.
func (a *App) askPort(in *bufio.Reader, label string, current, fallback int) (int, error) {
	def := current
	if def <= 0 {
		def = fallback
	}
	for range 3 {
		fmt.Fprintf(a.Stdout, "%s [%d]: ", label, def)
		v, more := readLine(in)
		if v == "" {
			return def, nil
		}
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 65535 {
			return n, nil
		}
		fmt.Fprintln(a.Stdout, "Escribe un puerto del 1 al 65535.")
		if !more {
			break
		}
	}
	return 0, fmt.Errorf("puerto inválido para %s", strings.ToLower(label))
}

// --- Lo común a avisar, reiniciar, levantar, relanzar-dp y vigia ---

// teamCtx es el proyecto y el equipo para los comandos que actúan sobre agentes
// que ya corren.
type teamCtx struct {
	root string
	cfg  Config
	hd   herdr
	ws   herdrWorkspace
}

// loadTeam encuentra el proyecto desde la carpeta actual y su workspace de herdr.
func (a *App) loadTeam(ctx context.Context) (*teamCtx, error) {
	cwd, err := a.Getwd()
	if err != nil {
		return nil, err
	}
	root := findRoot(cwd)
	cfg, err := loadConfig(root)
	if errors.Is(err, errNoConfig) {
		return nil, fmt.Errorf("no encuentro .pmt/config.yaml desde %s: corre el comando dentro del proyecto, después de «pmt pp» o «pmt equipo»", cwd)
	}
	if err != nil {
		return nil, err
	}
	if cfg.Herdr.Workspace == "" {
		cfg.Herdr.Workspace = filepath.Base(root)
	}
	hd, err := a.Herdr()
	if err != nil {
		return nil, err
	}
	ws, ok, err := findWorkspace(ctx, hd, cfg.Herdr.Workspace)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no existe el workspace %s en herdr: levanta al equipo con «pmt equipo»", cfg.Herdr.Workspace)
	}
	return &teamCtx{root: root, cfg: cfg, hd: hd, ws: ws}, nil
}

// member es un agente del equipo con su pane.
type member struct {
	info profileInfo
	conf ProfileConfig
	pane string
}

// name es su nombre en herdr, con el prefijo.
func (t *teamCtx) name(m member) string { return t.cfg.agentName(m.info.Agent) }

// paraUsuario es la ruta de .pmt/para-usuario.md.
func (t *teamCtx) paraUsuario() string { return filepath.Join(t.root, pmtDir, "para-usuario.md") }

// paneOf busca el pane de un agente en el workspace: el del agente con su
// nombre o, si no, el primero de la pestaña con su etiqueta. Vacío si no hay.
func paneOf(agents []herdrAgent, tabs []herdrTab, panes []herdrPane, ws, label, name string) string {
	for _, ag := range agents {
		if ag.Name == name && ag.Workspace == ws {
			for _, p := range panes {
				if p.ID == ag.Pane {
					return p.ID
				}
			}
		}
	}
	for _, t := range tabs {
		if t.Label != label {
			continue
		}
		for _, p := range panes {
			if p.Tab == t.ID {
				return p.ID
			}
		}
	}
	return ""
}

// member busca un agente por nombre. Falla si no está en la configuración o
// no tiene pestaña en el workspace.
func (t *teamCtx) member(ctx context.Context, arg string) (member, error) {
	info, ok := t.cfg.lookupAgent(arg)
	if !ok {
		return member{}, fmt.Errorf("agente desconocido: %s (usa pp, pc, tl, dp, devback, devfront, cr, qap o qa)", arg)
	}
	conf, ok := t.cfg.profile(info.Session)
	if !ok {
		return member{}, fmt.Errorf("%s no está en .pmt/config.yaml: configúralo con «pmt equipo %s»", info.Agent, info.Agent)
	}
	agents, err := t.hd.Agents(ctx)
	if err != nil {
		return member{}, err
	}
	tabs, err := t.hd.Tabs(ctx, t.ws.ID)
	if err != nil {
		return member{}, err
	}
	panes, err := t.hd.Panes(ctx, t.ws.ID)
	if err != nil {
		return member{}, err
	}
	pane := paneOf(agents, tabs, panes, t.ws.ID, info.Agent, t.cfg.agentName(info.Agent))
	if pane == "" {
		return member{}, fmt.Errorf("%s no corre en el workspace %s: levántalo con «pmt equipo %s»", info.Agent, t.ws.Label, info.Agent)
	}
	return member{info: info, conf: conf, pane: pane}, nil
}

// requireRunning falla si en el pane del agente no corre nada además del shell.
func (t *teamCtx) requireRunning(ctx context.Context, m member) error {
	busy, err := t.hd.Busy(ctx, m.pane)
	if err != nil {
		return err
	}
	if !busy {
		return fmt.Errorf("%s no corre en %s: levántalo con «pmt equipo %s»", m.info.Agent, m.pane, m.info.Agent)
	}
	return nil
}

// rename le pone al agente su nombre en herdr; un fallo solo se avisa.
func (a *App) rename(ctx context.Context, t *teamCtx, m member) {
	if err := t.hd.RenameAgent(ctx, m.pane, t.name(m)); err != nil {
		fmt.Fprintf(a.Stderr, "Aviso: no pude nombrar a %s: %s\n", t.name(m), err)
	}
}
