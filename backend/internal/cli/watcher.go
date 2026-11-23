package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// El vigía hace lo que vigia.sh del kit: observa a los agentes del proyecto y
// avisa al TL si todos llevan un rato sin trabajar. Tras un aviso no repite
// hasta que otro agente trabaje. Termina si alguien queda blocked o si hay una
// línea nueva en para-usuario.md.

const (
	watcherPulse = 15 * time.Second
	watcherQuiet = 150 * time.Second
)

func watcherPIDPath(root string) string { return filepath.Join(root, pmtDir, "vigia.pid") }
func watcherLogPath(root string) string { return filepath.Join(root, pmtDir, "vigia.log") }

// watcherPID es el pid guardado del vigía y si ese proceso sigue vivo.
func watcherPID(root string) (int, bool) {
	raw, err := os.ReadFile(watcherPIDPath(root))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	err = syscall.Kill(pid, 0)
	return pid, err == nil || errors.Is(err, syscall.EPERM)
}

// startWatcher arranca «pmt vigia» en segundo plano, en su propia sesión, con
// sus errores en .pmt/vigia.log.
func (a *App) startWatcher(root string) error {
	exe, err := a.Executable()
	if err != nil {
		return err
	}
	return a.Spawn(exe, []string{"vigia"}, root, watcherLogPath(root))
}

// spawn arranca un proceso desligado de la terminal (como setsid nohup), con la
// salida de errores al final de logPath, y no lo espera.
func spawn(bin string, args []string, dir, logPath string) error {
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stderr = log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// vigia es «pmt vigia [--pulso 15s] [--quieto 150s]».
func (a *App) vigia(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("vigia", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	pulse := fs.Duration("pulso", watcherPulse, "cada cuánto mira a los agentes")
	quiet := fs.Duration("quieto", watcherQuiet, "cuánto tiempo sin que nadie trabaje antes de avisar al TL")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 || *pulse <= 0 || *quiet <= 0 {
		return usageError{"uso: pmt vigia [--pulso 15s] [--quieto 150s]"}
	}
	t, err := a.loadTeam(ctx)
	if err != nil {
		return err
	}
	if pid, alive := watcherPID(t.root); alive && pid != os.Getpid() {
		return fmt.Errorf("el vigía ya corre (pid %d)", pid)
	}
	// pmt levantar lo usa para no arrancar un segundo vigía.
	pid := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(watcherPIDPath(t.root), []byte(pid+"\n"), 0o644); err != nil {
		return err
	}
	defer func() {
		if raw, err := os.ReadFile(watcherPIDPath(t.root)); err == nil && strings.TrimSpace(string(raw)) == pid {
			os.Remove(watcherPIDPath(t.root))
		}
	}()
	logFile, err := os.OpenFile(watcherLogPath(t.root), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	out := io.MultiWriter(a.Stdout, logFile)
	logf := func(format string, v ...any) {
		fmt.Fprintf(out, "%s %s\n", a.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, v...))
	}
	if err := touch(t.paraUsuario()); err != nil {
		return err
	}
	lines := func() int {
		raw, _ := os.ReadFile(t.paraUsuario())
		return strings.Count(string(raw), "\n")
	}

	logf("vigía en marcha (pid %s, workspace %s)", pid, t.ws.Label)
	start, quietFor, armed := lines(), time.Duration(0), true
	for {
		if err := a.Sleep(ctx, *pulse); err != nil {
			logf("vigía detenido")
			return nil
		}
		active, err := t.activeAgents(ctx)
		if err != nil {
			logf("no pude leer herdr: %s", err)
			continue
		}
		summary := strings.Join(active, " ")
		if strings.Contains(summary, ":blocked") {
			logf("BLOQUEADO %s", summary)
			return nil
		}
		if n := lines(); n != start {
			raw, _ := os.ReadFile(t.paraUsuario())
			all := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
			logf("AVISO\n%s", strings.Join(all[max(0, len(all)-3):], "\n"))
			return nil
		}
		working := false
		for _, s := range active {
			if strings.HasSuffix(s, ":working") {
				working = true
				// pp, pc y tl no rearman: hablar con el usuario o atender el aviso no cambia el flujo.
				switch strings.TrimSuffix(s, ":working") {
				case t.cfg.agentName("pp"), t.cfg.agentName("pc"), t.cfg.agentName("tl"):
				default:
					armed = true
				}
			}
		}
		if working {
			quietFor = 0
		} else {
			quietFor += *pulse
		}
		if quietFor >= *quiet && armed {
			logf("TODOS QUIETOS: aviso al TL")
			secs := int(quiet.Seconds())
			if err := a.notify(ctx, t, "tl", fmt.Sprintf("Vigía: todos llevan %d s quietos. Revisa el avance en PMT y levanta a quien le toque seguir; si nadie puede seguir, anótalo en para-usuario.md.", secs)); err != nil {
				logf("%s", err)
			}
			armed, quietFor = false, 0
		}
	}
}

// activeAgents son los agentes configurados que no están quietos, como
// «<nombre>:<estado>» (working o blocked).
func (t *teamCtx) activeAgents(ctx context.Context) ([]string, error) {
	agents, err := t.hd.Agents(ctx)
	if err != nil {
		return nil, err
	}
	tabs, err := t.hd.Tabs(ctx, t.ws.ID)
	if err != nil {
		return nil, err
	}
	panes, err := t.hd.Panes(ctx, t.ws.ID)
	if err != nil {
		return nil, err
	}
	byID := map[string]herdrPane{}
	for _, p := range panes {
		byID[p.ID] = p
	}
	var out []string
	for _, info := range allProfiles {
		if _, ok := t.cfg.profile(info.Session); !ok {
			continue
		}
		name := t.cfg.agentName(info.Agent)
		p, ok := byID[paneOf(agents, tabs, panes, t.ws.ID, info.Agent, name)]
		if !ok {
			continue
		}
		state := p.Status
		if strings.HasPrefix(p.Title, "Failed") {
			state = "failed"
		}
		switch state {
		case "", "idle", "done", "unknown":
			continue
		}
		out = append(out, name+":"+state)
	}
	return out, nil
}
