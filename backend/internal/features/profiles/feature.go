// Package profiles sirve el texto de los perfiles de los agentes: por REST y con
// la herramienta MCP get_profile. Las instrucciones del MCP de cada sesión son
// cortas y remiten a get_profile.
package profiles

import (
	"fmt"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/profiles/catalog"
	"pmt/internal/features/profiles/queries/get"
	"pmt/internal/features/profiles/queries/list"
	"pmt/internal/kernel"
)

// Register registra las rutas REST y la herramienta MCP de la feature.
func Register(mux *http.ServeMux, tools *sdk.Server) {
	list.New().Register(mux)
	profile := get.New()
	profile.Register(mux)
	profile.RegisterTool(tools)
}

// Instructions da las instrucciones MCP de una sesión: unas líneas que dicen su
// perfil y piden, como primer paso, leer el perfil completo con get_profile y
// después .pmt/entorno.md. Claude Code solo muestra los primeros ~4 000
// caracteres de las instrucciones y casi todos los perfiles miden más; por eso el
// texto completo va por get_profile.
//
// Sin perfil (User) no hay. Un token de agente (el kit) tampoco recibe: ya trae
// su perfil local completo y su proyecto no tiene .pmt/entorno.md.
func Instructions(actor kernel.Actor) (string, bool) {
	if actor.Agent {
		return "", false
	}
	entry, ok := catalog.ByProfile(actor.Profile)
	if !ok {
		return "", false
	}
	return fmt.Sprintf(`Esta sesión de PMT tiene el perfil %s (%s).

Antes de cualquier otra cosa:
1. Llama a get_profile: devuelve tu perfil completo, con tus reglas y las comunes. Síguelo en toda la sesión.
2. Lee %s en la raíz del proyecto: trae los datos de esta máquina (ruta, base, puertos, comandos del equipo) que tu perfil menciona.
`, entry.Profile, entry.Description, catalog.EnvFile), true
}
