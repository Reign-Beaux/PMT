package kernel

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// Profile es el perfil con el que actúa quien hace una petición.
type Profile string

// Perfiles de PMT. User es el usuario humano.
const (
	ProfilePP          Profile = "PP"
	ProfilePC          Profile = "PC"
	ProfileTL          Profile = "TL"
	ProfileDevFrontend Profile = "DevFrontend"
	ProfileDevBackend  Profile = "DevBackend"
	ProfileCR          Profile = "CR"
	ProfileQAP         Profile = "QAP"
	ProfileQA          Profile = "QA"
	ProfileDP          Profile = "DP"
	ProfileUser        Profile = "User"
)

// Actor identifica a quien ejecuta una operación. Lo resuelve el servidor y
// nunca llega como argumento. Un usuario actúa con su username y el perfil que
// declara su sesión, o User si no declara ninguno; un agente, con su nombre y su
// perfil fijo.
//
// UserID es el usuario cuya membresía de proyecto usa la operación: el propio
// usuario o, con un token de agente, el dueño de la instancia. Vacío, el actor no
// es miembro de ningún proyecto.
//
// UserName es el username de UserID: el mismo Name para un usuario, el del dueño
// de la instancia para un agente. Es el nombre de quien toma una etapa o un bug.
//
// Channel es la credencial con la que llegó: la cookie del sitio web o un token
// de API. Lo fija quien la autentica (rest.Authenticate, mcp.Authenticate).
//
// Agent indica que llegó con un token de agente (tabla agents, el kit) y no de
// usuario.
type Actor struct {
	Name     string
	Profile  Profile
	UserID   uuid.UUID
	UserName string
	Channel  Channel
	Agent    bool
}

// Channel es la credencial con la que llega una petición.
type Channel string

// Canales de una petición. El valor vacío cuenta como token: lo que solo se
// permite desde el sitio web exige ChannelSession explícito.
const (
	ChannelSession Channel = "session"
	ChannelToken   Channel = "token"
)

// InBrowser indica si el actor llegó con la cookie del sitio web.
func (a Actor) InBrowser() bool {
	return a.Channel == ChannelSession
}

// IsDev indica si el perfil es de desarrollo, sea frontend o backend.
func (p Profile) IsDev() bool {
	return p == ProfileDevFrontend || p == ProfileDevBackend
}

// Team es el equipo de una etapa: su TL responsable y sus devs, y el proyecto de
// la etapa, cuyos miembros owner o editor son los usuarios que pueden integrarlo.
type Team struct {
	Lead      string
	Members   []string
	ProjectID uuid.UUID
}

// IsLead indica si el actor es el TL de la etapa.
func (t Team) IsLead(actor Actor) bool {
	return t.Lead != "" && t.Lead == actor.Name
}

// HasMember indica si el agente o usuario pertenece al equipo de devs de la etapa.
func (t Team) HasMember(name string) bool {
	return slices.Contains(t.Members, name)
}

// ErrUnauthenticated indica que el token no corresponde a un agente ni a un
// usuario activos.
var ErrUnauthenticated = Unauthorized("token inválido")

// ErrSessionInvalid indica que la cookie no corresponde a una sesión vigente de
// un usuario activo.
var ErrSessionInvalid = Unauthorized("sesión no válida")

// Authenticator resuelve las credenciales de una petición a su actor: la cookie
// de sesión del navegador, que siempre actúa como User, o un token de API (de
// usuario o de agente) con el perfil que declara la sesión (vacío si no declara
// ninguno). Devuelve un error de tipo KindUnauthorized si las credenciales no son
// válidas y KindInvalid si el perfil declarado no cabe; cualquier otro es técnico.
type Authenticator interface {
	AuthenticateSession(ctx context.Context, session string) (Actor, error)
	AuthenticateToken(ctx context.Context, token string, declared Profile) (Actor, error)
}

// IsPlanner indica si el perfil planea: PP o el usuario, que tiene lo de PP.
func (p Profile) IsPlanner() bool {
	return p == ProfilePP || p == ProfileUser
}

// HasAnyTransition indica si el perfil hace alguna transición de la tabla. Sirve
// para rechazar, antes de leer el elemento, a un perfil que nunca mueve status; el
// resto sigue el orden normal: status, transición, perfil y ámbito.
func HasAnyTransition[S comparable](table map[S]map[S][]Profile, profile Profile) bool {
	for _, targets := range table {
		for _, profiles := range targets {
			if slices.Contains(profiles, profile) {
				return true
			}
		}
	}
	return false
}

// UserRef identifica a un usuario ante otras features: lo da auth a quien
// necesite resolver un username o un ID.
type UserRef struct {
	ID       uuid.UUID
	Username string
	Name     string
	Active   bool
}
