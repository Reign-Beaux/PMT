package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// tokenEnv es la variable con el token de PMT en el proceso del agente.
const tokenEnv = "PMT_TOKEN"

// mcpNamePattern es el nombre válido de un MCP extra: minúsculas y dígitos.
var mcpNamePattern = regexp.MustCompile(`^[a-z0-9]+$`)

// mcpTokenEnv es la variable con el token de un MCP extra.
func mcpTokenEnv(name string) string { return "MCP_TOKEN_" + strings.ToUpper(name) }

// launchSpec es lo que hace falta para lanzar un perfil: el comando, sus
// argumentos y las variables de entorno con secretos. Los tokens solo van en Env.
type launchSpec struct {
	Tool     string
	Args     []string
	Env      map[string]string
	Warnings []string // avisos de una línea para el usuario
}

// launchInput son los datos de un lanzamiento.
type launchInput struct {
	Config      Config
	Root        string
	Profile     string // nombre de sesión: pp, tl…
	Server      string
	Token       string
	ProfileFile string // .pmt/perfiles/<perfil>.md
	MCPFile     string // .pmt/mcp-<perfil>.json, solo Claude
	ProfileText string // el perfil, solo Codex
	Home        string
}

// mcpURL es la URL del MCP de PMT con el perfil de la sesión.
func mcpURL(server, profile string) string {
	return server + "/mcp?profile=" + profile
}

// extraMCP comprueba los MCP extra del perfil y lee sus tokens de ~/.config/pmt/.
func extraMCP(c Config, profile, home string) ([]string, map[string]string, error) {
	p, _ := c.profile(profile)
	tokens := map[string]string{}
	for _, name := range p.MCP {
		if !mcpNamePattern.MatchString(name) || name == "pmt" {
			return nil, nil, fmt.Errorf("MCP extra inválido para %s: %q (minúsculas y dígitos, distinto de pmt)", profile, name)
		}
		m, ok := c.MCP[name]
		if !ok || m.URL == "" {
			return nil, nil, fmt.Errorf("el perfil %s pide el MCP %q, que no está en mcp de .pmt/config.yaml", profile, name)
		}
		if m.Token == "" || filepath.IsAbs(m.Token) || strings.Contains(filepath.Clean(m.Token), "..") {
			return nil, nil, fmt.Errorf("el MCP %q necesita token: un archivo relativo a ~/.config/pmt/", name)
		}
		raw, err := os.ReadFile(filepath.Join(configDir(home), m.Token))
		if err != nil {
			return nil, nil, fmt.Errorf("no puedo leer el token del MCP %q en ~/.config/pmt/%s", name, m.Token)
		}
		tokens[name] = strings.TrimSpace(string(raw))
	}
	return p.MCP, tokens, nil
}

// buildLaunch arma el lanzamiento de Claude o Codex. Los tokens van en Env;
// los argumentos solo nombran las variables.
func buildLaunch(in launchInput) (launchSpec, error) {
	p, ok := in.Config.profile(in.Profile)
	if !ok {
		return launchSpec{}, fmt.Errorf("el perfil %s no está configurado en .pmt/config.yaml", in.Profile)
	}
	extras, extraTokens, err := extraMCP(in.Config, in.Profile, in.Home)
	if err != nil {
		return launchSpec{}, err
	}
	env := map[string]string{tokenEnv: in.Token}
	for name, tok := range extraTokens {
		env[mcpTokenEnv(name)] = tok
	}
	effort := p.Effort
	if effort == "" {
		effort = "medium"
	}

	switch p.Herramienta {
	case "claude":
		return launchSpec{Tool: "claude", Env: env, Args: []string{
			"--dangerously-skip-permissions", "--autocompact", "400k",
			"--model", p.Modelo, "--effort", effort,
			"--mcp-config", in.MCPFile, "--strict-mcp-config",
			"--append-system-prompt-file", in.ProfileFile,
		}}, nil
	case "codex":
		home := in.Config.codexHome(in.Home)
		env["CODEX_HOME"] = home
		stdio, file, err := codexStdioServers(home)
		if err != nil {
			return launchSpec{}, err
		}
		// Codex mezcla los -c con la tabla de su config.toml: una entrada stdio
		// con el mismo nombre queda con command y url, y Codex no arranca. Se
		// deshabilita esa entrada y el MCP va con otro nombre.
		var warnings []string
		rename := func(name string) string {
			if !stdio[name] {
				return name
			}
			warnings = append(warnings, codexStdioWarning(file, name))
			return name + codexRenameSuffix
		}
		pmtName := rename("pmt")
		instructions := in.ProfileText
		if pmtName != "pmt" {
			instructions += "\n\nEn esta sesión el MCP de PMT se llama `" + pmtName + "`, no `pmt`.\n"
		}
		args := []string{
			"-m", p.Modelo,
			"-c", "developer_instructions=" + tomlString(instructions),
			"-c", "model_reasoning_effort=" + tomlString(effort),
			// Como Claude con --dangerously-skip-permissions: sin preguntar por
			// comandos ni herramientas MCP, y sin depender del config.toml de quien
			// lo corra.
			"-c", `approval_policy="never"`,
			"-c", `sandbox_mode="danger-full-access"`,
			"-c", "allow_login_shell=false",
			"-c", "notice.hide_rate_limit_model_nudge=true",
		}
		if pmtName != "pmt" {
			args = append(args, "-c", "mcp_servers.pmt.enabled=false")
		}
		args = append(args,
			"-c", "mcp_servers."+pmtName+".url="+tomlString(mcpURL(in.Server, in.Profile)),
			"-c", "mcp_servers."+pmtName+".bearer_token_env_var="+tomlString(tokenEnv),
			// Como tabla en línea: la ruta como segmento entre comillas de una clave
			// con puntos («projects."/ruta".trust_level») no la toma Codex 0.171.
			"-c", "projects={"+tomlString(in.Config.projectPath(in.Root))+"={trust_level=\"trusted\"}}",
		)
		for _, name := range extras {
			as := rename(name)
			if as != name {
				args = append(args, "-c", "mcp_servers."+name+".enabled=false")
			}
			args = append(args,
				"-c", "mcp_servers."+as+".url="+tomlString(in.Config.MCP[name].URL),
				"-c", "mcp_servers."+as+".bearer_token_env_var="+tomlString(mcpTokenEnv(name)))
		}
		return launchSpec{Tool: "codex", Env: env, Args: args, Warnings: warnings}, nil
	default:
		return launchSpec{}, fmt.Errorf("herramienta desconocida para %s: %q (usa claude o codex)", in.Profile, p.Herramienta)
	}
}

// codexRenameSuffix va tras el nombre de un MCP que choca con una entrada
// stdio del config.toml de Codex.
const codexRenameSuffix = "_cli"

// codexStdioWarning es el aviso de una línea por una entrada stdio que choca.
func codexStdioWarning(file, name string) string {
	return fmt.Sprintf("aviso: %s tiene un MCP %q de tipo stdio; Codex lo tendrá deshabilitado y usará %q. Quita esa entrada vieja o usa codex_home en .pmt/config.yaml", file, name, name+codexRenameSuffix)
}

// codexStdioServers lee <CODEX_HOME>/config.toml y devuelve los nombres de
// mcp_servers de tipo stdio (con command). Sin archivo, ninguno.
func codexStdioServers(codexHome string) (map[string]bool, string, error) {
	file := filepath.Join(codexHome, "config.toml")
	raw, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, file, nil
	}
	if err != nil {
		return nil, file, fmt.Errorf("no puedo leer %s: %w", file, err)
	}
	var conf struct {
		MCPServers map[string]map[string]any `toml:"mcp_servers"`
	}
	if err := toml.Unmarshal(raw, &conf); err != nil {
		return nil, file, fmt.Errorf("no puedo leer %s como TOML: %v", file, err)
	}
	stdio := map[string]bool{}
	for name, server := range conf.MCPServers {
		if _, ok := server["command"]; ok {
			stdio[name] = true
		}
	}
	return stdio, file, nil
}

// tomlString es un texto como cadena TOML básica, que es la sintaxis de una
// cadena JSON salvo escapes que JSON no usa.
func tomlString(s string) string {
	raw, _ := json.Marshal(s)
	// json.Marshal escapa <, > y & como <…, válidos también en TOML.
	return string(raw)
}

// codexHome es CODEX_HOME: el de la config, con ~ expandido, o ~/.codex.
func (c Config) codexHome(home string) string {
	h := strings.TrimSpace(c.CodexHome)
	switch {
	case h == "" || h == "~":
		return filepath.Join(home, ".codex")
	case strings.HasPrefix(h, "~/"):
		return filepath.Join(home, h[2:])
	}
	return h
}

// mcpConfigJSON es .pmt/mcp-<perfil>.json para Claude: el MCP de PMT con el
// perfil y los extra. Los tokens quedan como ${VARIABLE}, que Claude expande
// del entorno: nunca se escriben aquí.
func mcpConfigJSON(c Config, server, profile string) ([]byte, error) {
	servers := map[string]any{
		"pmt": map[string]any{
			"type":    "http",
			"url":     mcpURL(server, profile),
			"headers": map[string]string{"Authorization": "Bearer ${" + tokenEnv + ":-}"},
		},
	}
	p, _ := c.profile(profile)
	for _, name := range p.MCP {
		m := c.MCP[name]
		servers[name] = map[string]any{
			"type":    "http",
			"url":     m.URL,
			"headers": map[string]string{"Authorization": "Bearer ${" + mcpTokenEnv(name) + ":-}"},
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// shellQuote cita un argumento para sh.
func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("/._-+:=@", r))
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// envList agrega env al entorno actual, reemplazando lo que ya estuviera.
func envList(base []string, env map[string]string) []string {
	out := make([]string, 0, len(base)+len(env))
	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if _, ok := env[k]; ok {
			continue
		}
		out = append(out, kv)
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}
