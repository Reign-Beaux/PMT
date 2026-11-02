// Package domain contiene el agregado Agent.
package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrAgentNameRequired indica que el nombre está vacío.
	ErrAgentNameRequired = kernel.Invalid("el nombre del agente es obligatorio")
	// ErrAgentProfileInvalid indica que el perfil no es uno de los conocidos.
	ErrAgentProfileInvalid = kernel.Invalid("perfil desconocido: usa PP, PC, TL, DevFrontend, DevBackend, CR, QAP, QA, DP o User")
	// ErrAgentNameTaken indica que ya existe un agente con ese nombre.
	ErrAgentNameTaken = kernel.Conflict("ya existe un agente con ese nombre")
	// ErrAgentNotFound indica que el agente no existe.
	ErrAgentNotFound = kernel.NotFound("agente no encontrado")
	// ErrAgentRevoked indica que el agente está revocado.
	ErrAgentRevoked = kernel.Conflict("el agente está revocado")
)

var profiles = []kernel.Profile{
	kernel.ProfilePP, kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDevFrontend,
	kernel.ProfileDevBackend, kernel.ProfileCR, kernel.ProfileQAP, kernel.ProfileQA, kernel.ProfileDP, kernel.ProfileUser,
}

// Agent es quien actúa en PMT: un agente o el usuario humano. Su perfil no cambia
// y no se borra: se revoca, porque el historial conserva su nombre.
type Agent struct {
	ID        uuid.UUID
	Name      string
	Profile   kernel.Profile
	RevokedAt *time.Time
	CreatedAt time.Time
}

// NewAgent crea un agente válido.
func NewAgent(name string, profile kernel.Profile) (Agent, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Agent{}, ErrAgentNameRequired
	}
	if !slices.Contains(profiles, profile) {
		return Agent{}, ErrAgentProfileInvalid
	}
	return Agent{ID: uuid.New(), Name: name, Profile: profile}, nil
}

// Revoked indica si el agente está revocado.
func (a Agent) Revoked() bool {
	return a.RevokedAt != nil
}

// Actor devuelve la identidad con la que actúa.
func (a Agent) Actor() kernel.Actor {
	return kernel.Actor{Name: a.Name, Profile: a.Profile}
}
