package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Los comandos del equipo hacen lo que hacían avisar.sh, reiniciar.sh,
// levantar.sh y relanzar-dp.sh del kit, con los agentes de .pmt/config.yaml.

// --- pmt avisar ---

// avisar es «pmt avisar <agente> "mensaje"».
func (a *App) avisar(ctx context.Context, args []string) error {
	if len(args) < 2 || strings.TrimSpace(strings.Join(args[1:], " ")) == "" {
		return usageError{`uso: pmt avisar <agente> "mensaje"`}
	}
	t, err := a.loadTeam(ctx)
	if err != nil {
		return err
	}
	return a.notify(ctx, t, args[0], strings.Join(args[1:], " "))
}

// composerLine es la línea del compositor: el prompt de Claude (❯) o de Codex (›).
var composerLine = regexp.MustCompile(`^[›❯]( |$)`)

// look es lo que se ve del aviso en el pane.
type look struct {
	pending bool // el aviso sigue en el compositor, sin enviar
	busy    bool // el agente trabaja
	count   int  // veces que se ve arriba del compositor; -1 sin compositor
}

// observe lee el pane como mirar() de avisar.sh. El compositor es la última
// línea con el prompt del agente: lo de ahí para abajo no se ha enviado; lo de
// arriba, enviado o en cola, ya lo tiene.
func observe(p herdrPane, screen, msg string) look {
	busy := p.Status == "working" || p.Status == "blocked" || strings.HasPrefix(p.Title, "Working")
	if strings.Contains(p.Title, "Failed") {
		busy = false
	}
	norm := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	m := []rune(norm(msg))
	frag, tail := string(m[:min(24, len(m))]), string(m[max(0, len(m)-24):])
	var lines []string
	for l := range strings.SplitSeq(screen, "\n") {
		lines = append(lines, norm(l))
	}
	c := -1
	for i, l := range lines {
		if composerLine.MatchString(l) {
			c = i
		}
	}
	if c < 0 {
		return look{busy: busy, count: -1}
	}
	below := strings.Join(lines[c:], " ")
	above := strings.Join(lines[:c], " ")
	return look{
		pending: strings.Contains(below, frag) || strings.Contains(below, "[Pasted"),
		busy:    busy,
		count:   strings.Count(above, frag) + strings.Count(above, tail),
	}
}

// notify envía el aviso y comprueba que llegó. Si no se confirma, reintenta una
// vez sin duplicarlo; si sigue sin confirmarse, deja una línea en
// para-usuario.md y devuelve error.
func (a *App) notify(ctx context.Context, t *teamCtx, agent, text string) error {
	msg := strings.ReplaceAll(text, "\n", " ")
	m, err := t.member(ctx, agent)
	if err != nil {
		return err
	}
	hd, pane := t.hd, m.pane
	fail := func(reason string) error {
		line := fmt.Sprintf("%s aviso sin confirmar a %s: %s\n", a.Now().Format("2006-01-02 15:04:05"), m.info.Agent, msg)
		if f, err := os.OpenFile(t.paraUsuario(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			f.WriteString(line)
			f.Close()
		}
		return fmt.Errorf("aviso sin confirmar a %s (%s)", m.info.Agent, reason)
	}
	see := func() look {
		p, err := hd.Pane(ctx, pane)
		if err != nil {
			return look{count: -1}
		}
		screen, err := hd.Read(ctx, pane, 80)
		if err != nil {
			return look{count: -1}
		}
		return observe(p, screen, msg)
	}
	// En Codex, un $ puede abrir el selector de menciones, donde Enter elige una
	// y cambia el mensaje. Esc solo si está abierto: si no, interrumpiría el turno.
	closeSelector := func() {
		screen, err := hd.Read(ctx, pane, 12)
		if err != nil || !strings.Contains(screen, "esc close") {
			return
		}
		hd.SendKeys(ctx, pane, "esc")
		a.Sleep(ctx, time.Second)
	}
	send := func() bool {
		if p, err := hd.Pane(ctx, pane); err != nil || p.Status == "blocked" {
			return false
		}
		if m.conf.Herramienta == "codex" && strings.Contains(msg, "$") {
			// agent prompt manda el Enter junto con el texto: aquí van por separado.
			if hd.SendText(ctx, pane, msg) != nil {
				return false
			}
			a.Sleep(ctx, time.Second)
			closeSelector()
			return hd.SendKeys(ctx, pane, "enter") == nil
		}
		return hd.Prompt(ctx, pane, msg) == nil
	}

	// Sin programa en primer plano, el mensaje correría en el shell.
	if busy, err := hd.Busy(ctx, pane); err != nil || !busy {
		return fail("sin agente en " + pane)
	}
	first := see()
	if first.count < 0 {
		return fail("sin agente listo en " + pane)
	}
	var last look
	// confirm espera la confirmación, unos 4 s.
	confirm := func() bool {
		for range 8 {
			last = see()
			if !last.pending && last.count > first.count {
				return true
			}
			if !last.pending && last.busy && !first.busy {
				return true
			}
			if a.Sleep(ctx, 500*time.Millisecond) != nil {
				return false
			}
		}
		return false
	}
	if !send() {
		return fail("no se pudo enviar")
	}
	if confirm() {
		return nil
	}
	if last.pending {
		closeSelector()
		hd.SendKeys(ctx, pane, "enter")
	} else {
		// Ocupado y sin rastro: pudo entrar. Reenviar podría duplicarlo.
		if last.busy {
			return fail("ocupado y sin rastro del mensaje")
		}
		if !send() {
			return fail("no se pudo reenviar")
		}
	}
	if confirm() {
		return nil
	}
	// Lo que quede en el compositor se pegaría al siguiente aviso.
	if last.pending {
		hd.SendKeys(ctx, pane, "ctrl+u")
	}
	return fail("no confirmado tras reintentar")
}

// --- pmt reiniciar, levantar y relanzar-dp ---

// reiniciar es «pmt reiniciar <agente>».
func (a *App) reiniciar(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return usageError{"uso: pmt reiniciar <agente>"}
	}
	t, err := a.loadTeam(ctx)
	if err != nil {
		return err
	}
	m, err := a.restart(ctx, t, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "%s reiniciado\n", m.info.Agent)
	return nil
}

// restart deja al agente con una sesión vacía: /new en Claude; en Codex
// relanza el proceso, porque tras /new descarta el primer mensaje.
func (a *App) restart(ctx context.Context, t *teamCtx, agent string) (member, error) {
	m, err := t.member(ctx, agent)
	if err != nil {
		return m, err
	}
	if err := t.requireRunning(ctx, m); err != nil {
		return m, err
	}
	// Quien avisa al terminar puede seguir cerrando su turno: se le espera hasta
	// 90 s. Si no basta, quien reinicia vuelve a intentarlo: el aviso ya llegó.
	for i := 0; ; i++ {
		p, err := t.hd.Pane(ctx, m.pane)
		if err != nil {
			return m, err
		}
		if p.Status != "working" || strings.Contains(p.Title, "Failed") {
			break
		}
		if i >= 30 {
			return m, fmt.Errorf("%s sigue trabajando tras 90 s: si ya te avisó, vuelve a intentarlo; si no, espera su aviso", m.info.Agent)
		}
		if err := a.Sleep(ctx, 3*time.Second); err != nil {
			return m, err
		}
	}
	if m.conf.Herramienta == "codex" {
		return m, a.relaunch(ctx, t, m)
	}
	if err := t.hd.SendKeys(ctx, m.pane, "ctrl+u"); err != nil {
		return m, err
	}
	if err := t.hd.Run(ctx, m.pane, "/new"); err != nil {
		return m, err
	}
	a.Sleep(ctx, 4*time.Second)
	a.rename(ctx, t, m)
	return m, nil
}

// relaunch cierra el programa del agente y lo vuelve a lanzar con sesión
// vacía, con el perfil recién descargado y la configuración actual.
func (a *App) relaunch(ctx context.Context, t *teamCtx, m member) error {
	creds, err := loadCredentials(a.Home)
	if err != nil {
		return err
	}
	quit := "/exit"
	if m.conf.Herramienta == "codex" {
		quit = "/quit"
	}
	if err := t.hd.SendKeys(ctx, m.pane, "ctrl+u"); err != nil {
		return err
	}
	if err := t.hd.Run(ctx, m.pane, quit); err != nil {
		return err
	}
	closed := false
	for range 10 {
		busy, err := t.hd.Busy(ctx, m.pane)
		if err != nil {
			return err
		}
		if !busy {
			closed = true
			break
		}
		if err := a.Sleep(ctx, time.Second); err != nil {
			return err
		}
	}
	if !closed {
		return fmt.Errorf("%s no cerró", m.info.Agent)
	}
	if err := a.prepareLaunch(ctx, a.client(creds.Server, creds.Token), t.cfg, t.root, creds.Server, m.info.Session); err != nil {
		return err
	}
	if err := a.runLauncher(ctx, t.hd, m.pane, t.root, m.info.Session); err != nil {
		return err
	}
	// herdr tarda en olvidar al agente que salió: se le da tiempo antes de buscar al nuevo.
	a.Sleep(ctx, 5*time.Second)
	if !a.waitAgent(ctx, t.hd, t.ws.ID, m.pane) {
		return fmt.Errorf("herdr no detecta a %s tras relanzarlo: revisa su pestaña", m.info.Agent)
	}
	a.rename(ctx, t, m)
	return nil
}

// levantar es «pmt levantar <agente> "encargo"»: reinicia y da el encargo.
func (a *App) levantar(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return usageError{`uso: pmt levantar <agente> "encargo"`}
	}
	task := strings.TrimSpace(strings.Join(args[1:], " "))
	if task == "" {
		return errors.New("falta el encargo")
	}
	t, err := a.loadTeam(ctx)
	if err != nil {
		return err
	}
	m, err := a.restart(ctx, t, args[0])
	if err != nil {
		return err
	}
	// Con el TL arranca el vigía, si no corre ya: termina solo cuando alguien
	// anota en para-usuario.md.
	if m.info.Agent == "tl" {
		if _, alive := watcherPID(t.root); !alive {
			if err := touch(t.paraUsuario()); err != nil {
				return err
			}
			if err := a.startWatcher(t.root); err != nil {
				fmt.Fprintf(a.Stderr, "Aviso: no pude arrancar el vigía: %s. Arráncalo con «pmt vigia».\n", err)
			}
		}
	}
	if err := a.notify(ctx, t, m.info.Agent, strings.Join(args[1:], " ")); err != nil {
		return err
	}
	a.Sleep(ctx, 6*time.Second)
	a.rename(ctx, t, m)
	fmt.Fprintf(a.Stdout, "%s levantado\n", m.info.Agent)
	return nil
}

// relanzarDP es «pmt relanzar-dp <medium|high>»: guarda el effort en
// .pmt/config.yaml, para que se conserve al reiniciar, y relanza a DP.
func (a *App) relanzarDP(ctx context.Context, args []string) error {
	if len(args) != 1 || (args[0] != "medium" && args[0] != "high") {
		return usageError{"uso: pmt relanzar-dp <medium|high>"}
	}
	t, err := a.loadTeam(ctx)
	if err != nil {
		return err
	}
	m, err := t.member(ctx, "dp")
	if err != nil {
		return err
	}
	if err := t.requireRunning(ctx, m); err != nil {
		return err
	}
	m.conf.Effort = args[0]
	t.cfg.Perfiles[m.info.Session] = m.conf
	if err := saveConfig(t.root, t.cfg); err != nil {
		return err
	}
	if err := a.relaunch(ctx, t, m); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "dp relanzado en %s\n", args[0])
	return nil
}

// touch crea el archivo si no existe.
func touch(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
