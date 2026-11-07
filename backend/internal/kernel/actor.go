package kernel

import (
	"context"
	"errors"
	"slices"
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
// nunca llega como argumento.
type Actor struct {
	Name    string
	Profile Profile
}

// Owner es el usuario humano: el actor de toda petición sin token. No tiene fila
// en agents, por lo que no puede ser TL, miembro de un equipo ni dueño de una tarea.
var Owner = Actor{Name: "saul", Profile: ProfileUser}

// IsDev indica si el perfil es de desarrollo, sea frontend o backend.
func (p Profile) IsDev() bool {
	return p == ProfileDevFrontend || p == ProfileDevBackend
}

// Team es el equipo de una etapa: su TL responsable y sus devs.
type Team struct {
	Lead    string
	Members []string
}

// IsLead indica si el actor es el TL de la etapa.
func (t Team) IsLead(actor Actor) bool {
	return t.Lead != "" && t.Lead == actor.Name
}

// HasMember indica si el agente pertenece al equipo de devs de la etapa.
func (t Team) HasMember(name string) bool {
	return slices.Contains(t.Members, name)
}

// ErrUnauthenticated indica que el token no corresponde a un agente activo.
var ErrUnauthenticated = errors.New("token inválido")

// Authenticator resuelve un token a su actor. Devuelve ErrUnauthenticated si el
// token no es válido; cualquier otro error es técnico.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (Actor, error)
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
