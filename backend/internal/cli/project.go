package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Carpeta local de pmt en la raíz de cada proyecto. Va fuera de git.
const pmtDir = ".pmt"

// Config es .pmt/config.yaml: lo que pmt sabe del proyecto en esta máquina.
// No lleva tokens: el de PMT sale de las credenciales y los de MCP extra de
// archivos en ~/.config/pmt/.
type Config struct {
	Version  int            `yaml:"version"`
	Proyecto ProjectConfig  `yaml:"proyecto"`
	Base     DatabaseConfig `yaml:"base"`
	Puertos  PortsConfig    `yaml:"puertos"`
	Herdr    HerdrConfig    `yaml:"herdr"`
	// CodexHome es la carpeta de Codex (CODEX_HOME). Vacía: ~/.codex.
	CodexHome string `yaml:"codex_home"`
	// MCP son servidores MCP extra que pueden cargar los perfiles.
	MCP map[string]MCPConfig `yaml:"mcp,omitempty"`
	// Perfiles por nombre de sesión (pp, tl, devbackend…).
	Perfiles map[string]ProfileConfig `yaml:"perfiles"`
}

// ProjectConfig es el proyecto de PMT y lo que los agentes leen de él.
type ProjectConfig struct {
	// ID del proyecto en PMT. Vacío mientras PP no lo crea.
	ID     string `yaml:"id"`
	Nombre string `yaml:"nombre"`
	// NuevoDesde es cuándo se eligió «nuevo, lo crea PP»: pmt busca proyectos
	// del usuario creados después para guardarlo.
	NuevoDesde *time.Time `yaml:"nuevo_desde,omitempty"`
	// Ruta del proyecto. Vacía: la raíz donde está .pmt/.
	Ruta        string `yaml:"ruta"`
	Descripcion string `yaml:"descripcion"`
}

// DatabaseConfig es la base del proyecto. La pregunta pmt equipo.
type DatabaseConfig struct {
	Cadena   string `yaml:"cadena"`
	Consulta string `yaml:"consulta"`
}

// PortsConfig son los puertos del proyecto. Los pregunta pmt equipo.
type PortsConfig struct {
	Backend  int `yaml:"backend"`
	Frontend int `yaml:"frontend"`
}

// HerdrConfig es dónde corre el equipo en herdr.
type HerdrConfig struct {
	Workspace string `yaml:"workspace"`
	// Prefijo de los nombres de los agentes, si choca con otro equipo vivo.
	Prefijo string `yaml:"prefijo"`
}

// MCPConfig es un servidor MCP extra (HTTP con Bearer).
type MCPConfig struct {
	URL string `yaml:"url"`
	// Token es el archivo con el token, relativo a ~/.config/pmt/.
	Token string `yaml:"token"`
}

// ProfileConfig es cómo se lanza un perfil.
type ProfileConfig struct {
	Herramienta string   `yaml:"herramienta"`
	Modelo      string   `yaml:"modelo"`
	Effort      string   `yaml:"effort"`
	MCP         []string `yaml:"mcp,omitempty"`
}

const configHeader = `# Configuración de pmt para este proyecto en esta máquina. Va fuera de git.
# No lleva tokens: el de PMT sale de «pmt login» y los de MCP extra de archivos en ~/.config/pmt/.
# La generan «pmt pp» y «pmt equipo»; con --configurar vuelven a preguntar. Ver .docs/perfiles.md en PMT.
`

// findRoot es la carpeta más cercana hacia arriba desde dir que ya tiene .pmt/;
// si ninguna la tiene, dir.
func findRoot(dir string) string {
	for d := dir; ; {
		if info, err := os.Stat(filepath.Join(d, pmtDir)); err == nil && info.IsDir() {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}

func configPath(root string) string { return filepath.Join(root, pmtDir, "config.yaml") }

// errNoConfig indica que el proyecto aún no tiene .pmt/config.yaml.
var errNoConfig = errors.New("el proyecto no tiene .pmt/config.yaml")

// loadConfig lee .pmt/config.yaml.
func loadConfig(root string) (Config, error) {
	raw, err := os.ReadFile(configPath(root))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, errNoConfig
	}
	if err != nil {
		return Config{}, fmt.Errorf("leer %s: %w", configPath(root), err)
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("%s dañado: %w", configPath(root), err)
	}
	return c, nil
}

// saveConfig escribe .pmt/config.yaml, creando .pmt/.
func saveConfig(root string, c Config) error {
	c.Version = 1
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	raw := buf.Bytes()
	if err := os.MkdirAll(filepath.Join(root, pmtDir), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(configPath(root), append([]byte(configHeader), raw...), 0o644)
}

// writeFileAtomic escribe a un temporal en la misma carpeta y lo renombra.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("escribir %s: %w", path, err)
	}
	return nil
}

// projectPath es la ruta del proyecto: la de la config o la raíz.
func (c Config) projectPath(root string) string {
	if c.Proyecto.Ruta != "" {
		return c.Proyecto.Ruta
	}
	return root
}

// profile devuelve la configuración de un perfil y si existe.
func (c Config) profile(name string) (ProfileConfig, bool) {
	p, ok := c.Perfiles[name]
	return p, ok && p.Herramienta != ""
}

// isGitRepo indica si root está dentro de un repositorio git.
func isGitRepo(root string) bool {
	for d := root; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return false
		}
		d = parent
	}
}

// ensureGitignore agrega .pmt/ al .gitignore de root, creándolo si no existe.
// No duplica la línea si ya está (como .pmt, .pmt/, /.pmt o /.pmt/).
// Devuelve si lo cambió.
func ensureGitignore(root string) (bool, error) {
	path := filepath.Join(root, ".gitignore")
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("leer %s: %w", path, err)
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		switch strings.TrimSpace(line) {
		case ".pmt", ".pmt/", "/.pmt", "/.pmt/":
			return false, nil
		}
	}
	text := string(raw)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += pmtDir + "/\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return false, fmt.Errorf("escribir %s: %w", path, err)
	}
	return true, nil
}
