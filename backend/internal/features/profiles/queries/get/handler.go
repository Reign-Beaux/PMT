// Package get consulta el texto servido de un perfil.
package get

import (
	"pmt/internal/features/profiles/catalog"
	"pmt/internal/kernel"
)

// ErrProfileNotFound indica que el perfil pedido no existe en el catálogo.
var ErrProfileNotFound = kernel.NotFound("perfil no encontrado: usa pp, pc, tl, dp, devbackend, devfrontend, cr, qap o qa")

// Handler ejecuta la operación.
type Handler struct{}

// New construye la operación.
func New() *Handler { return &Handler{} }

// Handle devuelve el texto servido del perfil por su nombre de sesión, el mismo
// que recibe en las instrucciones del MCP una sesión con ese perfil.
func (h *Handler) Handle(name string) (string, error) {
	entry, ok := catalog.ByName(name)
	if !ok {
		return "", ErrProfileNotFound
	}
	text, _ := catalog.Served(entry.Profile)
	return text, nil
}

// HandleActor devuelve el texto servido del perfil de la sesión. User, sin perfil
// declarado, no tiene: responde ErrNoProfile.
func (h *Handler) HandleActor(actor kernel.Actor) (string, error) {
	text, ok := catalog.Served(actor.Profile)
	if !ok {
		return "", ErrNoProfile
	}
	return text, nil
}
