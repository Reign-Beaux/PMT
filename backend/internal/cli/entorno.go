package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// profileInfo es un perfil de PMT: su nombre de sesión (?profile=), el del
// agente en herdr y el que muestra la tabla.
type profileInfo struct {
	Session string
	Agent   string
	Label   string
}

// allProfiles son los nueve perfiles en el orden del kit. El nombre en herdr
// es el del kit (devback, devfront), que los perfiles mencionan.
var allProfiles = []profileInfo{
	{"pp", "pp", "PP"},
	{"pc", "pc", "PC"},
	{"tl", "tl", "TL"},
	{"dp", "dp", "DP"},
	{"devbackend", "devback", "DevBackend"},
	{"devfrontend", "devfront", "DevFrontend"},
	{"cr", "cr", "CR"},
	{"qap", "qap", "QAP"},
	{"qa", "qa", "QA"},
}

// pending es lo que dice entorno.md de un dato que aún no se sabe.
const pending = "pendiente: lo completa `pmt equipo`"

// agentName es el nombre del agente en herdr, con el prefijo de la config.
func (c Config) agentName(agent string) string {
	return c.Herdr.Prefijo + agent
}

// renderEntorno arma .pmt/entorno.md con el formato de .docs/perfiles.md.
// username es el de quien lanza: todos sus agentes actúan con él en PMT.
func renderEntorno(c Config, root, username string) string {
	path := c.projectPath(root)
	orPending := func(v string) string {
		if strings.TrimSpace(v) == "" {
			return pending
		}
		return v
	}
	code := func(v string) string {
		if strings.TrimSpace(v) == "" {
			return pending
		}
		return "`" + v + "`"
	}
	port := func(p int) string {
		if p <= 0 {
			return pending
		}
		return strconv.Itoa(p)
	}

	var b strings.Builder
	b.WriteString("# Entorno del proyecto\n\n")
	b.WriteString("Lo genera `pmt pp` en cada arranque desde `.pmt/config.yaml`: no lo edites a mano.\n\n")

	b.WriteString("## Proyecto\n")
	fmt.Fprintf(&b, "- Ruta: `%s`\n", path)
	fmt.Fprintf(&b, "- Descripción: %s\n", orPending(c.Proyecto.Descripcion))
	if c.Proyecto.ID != "" {
		fmt.Fprintf(&b, "- Proyecto en PMT: %s (`%s`)\n", c.Proyecto.Nombre, c.Proyecto.ID)
	} else {
		b.WriteString("- Proyecto en PMT: todavía no existe; lo crea PP\n")
	}

	b.WriteString("\n## Base de datos\n")
	if c.Base.Cadena != "" {
		fmt.Fprintf(&b, "- Cadena: `%s` (solo DevBackend la escribe, en `config/config.dev.yaml`; los demás usan ese archivo)\n", c.Base.Cadena)
	} else {
		fmt.Fprintf(&b, "- Cadena: %s\n", pending)
	}
	if c.Base.Consulta != "" {
		fmt.Fprintf(&b, "- Consultar: `%s` (cambia `<consulta>` por la tuya)\n", c.Base.Consulta)
	} else {
		fmt.Fprintf(&b, "- Consultar: %s\n", pending)
	}

	b.WriteString("\n## Puertos\n")
	fmt.Fprintf(&b, "- Backend: %s\n", port(c.Puertos.Backend))
	fmt.Fprintf(&b, "- Frontend: %s\n", port(c.Puertos.Frontend))

	b.WriteString("\n## Comandos del equipo\n")
	b.WriteString("- Avisar: `pmt avisar <agente> \"mensaje\"`\n")
	b.WriteString("- Levantar: `pmt levantar <agente> \"encargo\"`\n")
	b.WriteString("- Reiniciar: `pmt reiniciar <agente>`\n")
	b.WriteString("- Relanzar a DP: `pmt relanzar-dp <medium|high>`\n")
	b.WriteString("- Vigía: `pmt vigia`\n")

	b.WriteString("\n## Para el usuario\n")
	fmt.Fprintf(&b, "- Archivo: %s\n", code(filepath.Join(root, pmtDir, "para-usuario.md")))

	b.WriteString("\n## Agentes\n")
	b.WriteString("| Agente | PMT | Perfil |\n|---|---|---|\n")
	for _, p := range allProfiles {
		if _, ok := c.profile(p.Session); !ok {
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c.agentName(p.Agent), username, p.Label)
	}
	return b.String()
}
