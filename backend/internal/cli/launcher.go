package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// launcherCommand es el subcomando interno que pmt pp y pmt equipo escriben en el pane del
// agente. Corre dentro del pane, como lanzar.sh del kit: lee el token de las
// credenciales, lo pone en el entorno y se reemplaza por Claude o Codex. Así el
// token no aparece en el comando escrito en el pane, ni en sus argumentos (ps),
// ni en archivos del proyecto.
const launcherCommand = "_lanzar"

// launch es «pmt _lanzar <perfil> <raíz>».
func (a *App) launch(_ context.Context, args []string) error {
	if len(args) != 2 {
		return usageError{"uso interno: pmt " + launcherCommand + " <perfil> <raíz del proyecto>"}
	}
	profile, root := args[0], args[1]
	creds, err := loadCredentials(a.Home)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}
	p, ok := cfg.profile(profile)
	if !ok {
		return fmt.Errorf("el perfil %s no está configurado en %s", profile, configPath(root))
	}
	in := launchInput{
		Config:      cfg,
		Root:        root,
		Profile:     profile,
		Server:      creds.Server,
		Token:       creds.Token,
		ProfileFile: filepath.Join(root, pmtDir, "perfiles", profile+".md"),
		MCPFile:     filepath.Join(root, pmtDir, "mcp-"+profile+".json"),
		Home:        a.Home,
	}
	text, err := os.ReadFile(in.ProfileFile)
	if err != nil {
		return fmt.Errorf("falta el perfil descargado %s: corre «pmt equipo»", in.ProfileFile)
	}
	in.ProfileText = string(text)
	if p.Herramienta == "claude" {
		if _, err := os.Stat(in.MCPFile); err != nil {
			return fmt.Errorf("falta %s: corre «pmt equipo»", in.MCPFile)
		}
	}
	spec, err := buildLaunch(in)
	if err != nil {
		return err
	}
	bin, err := a.LookPath(spec.Tool)
	if err != nil {
		return fmt.Errorf("no encuentro %s en el PATH", spec.Tool)
	}
	if err := os.Chdir(cfg.projectPath(root)); err != nil {
		return err
	}
	return a.Exec(bin, append([]string{spec.Tool}, spec.Args...), envList(a.Environ(), spec.Env))
}
