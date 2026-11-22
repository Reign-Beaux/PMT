package kernel

import (
	"fmt"
	"strings"
)

// ErrProfileUnknown indica que la sesión declaró un perfil que no existe.
var ErrProfileUnknown = Invalid("perfil desconocido: usa pp, pc, tl, dp, devfrontend, devbackend, cr, qap, qa o user")

// declarable son los perfiles que una sesión puede declarar, por su nombre en
// minúsculas. user equivale a no declarar.
var declarable = map[string]Profile{
	"pp":          ProfilePP,
	"pc":          ProfilePC,
	"tl":          ProfileTL,
	"dp":          ProfileDP,
	"devfrontend": ProfileDevFrontend,
	"devbackend":  ProfileDevBackend,
	"cr":          ProfileCR,
	"qap":         ProfileQAP,
	"qa":          ProfileQA,
	"user":        "",
}

// ParseProfile interpreta el perfil que declara una sesión, sin distinguir
// mayúsculas. Vacío o user devuelven el perfil vacío: la sesión no declara
// ninguno. Un valor desconocido devuelve ErrProfileUnknown.
func ParseProfile(value string) (Profile, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	profile, ok := declarable[strings.ToLower(value)]
	if !ok {
		return "", ErrProfileUnknown
	}
	return profile, nil
}

// AsUser es el actor de un usuario que declara un perfil: su username con ese
// perfil, o con User si no declara ninguno. Un usuario puede usar cualquiera.
func AsUser(username string, declared Profile) Actor {
	if declared == "" {
		declared = ProfileUser
	}
	return Actor{Name: username, Profile: declared}
}

// AsAgent comprueba el perfil que declara la sesión de un agente: su perfil es
// fijo, así que solo acepta no declarar ninguno o declarar el suyo.
func AsAgent(agent Actor, declared Profile) (Actor, error) {
	if declared != "" && declared != agent.Profile {
		return Actor{}, Invalid(fmt.Sprintf("el token es del agente %s con perfil %s: declara ese perfil o ninguno", agent.Name, agent.Profile))
	}
	return agent, nil
}
