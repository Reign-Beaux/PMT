package kernel

import (
	"context"
	"errors"
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
	ProfileUser        Profile = "User"
)

// Actor identifica a quien ejecuta una operación. Lo resuelve el servidor y
// nunca llega como argumento.
type Actor struct {
	Name    string
	Profile Profile
}

// Anonymous es el actor de una petición sin token, admitida solo durante la
// transición a los tokens.
var Anonymous = Actor{Name: "sin-autenticar"}

// ErrUnauthenticated indica que el token no corresponde a un agente activo.
var ErrUnauthenticated = errors.New("token inválido")

// Authenticator resuelve un token a su actor. Devuelve ErrUnauthenticated si el
// token no es válido; cualquier otro error es técnico.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (Actor, error)
}
