// Package lookup consulta el perfil de un agente activo.
package lookup

import (
	"context"
	"errors"
	"strings"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	GetByName(ctx context.Context, name string) (domain.Agent, error)
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el perfil del agente. Un agente revocado no se puede asignar.
func (h *Handler) Handle(ctx context.Context, name string) (kernel.Profile, error) {
	agent, err := h.reader.GetByName(ctx, name)
	if err != nil {
		return "", err
	}
	if agent.RevokedAt != nil {
		return "", domain.ErrAgentRevoked
	}
	return agent.Profile, nil
}

// Exists indica si hay un agente con ese nombre, revocado o no: su nombre sigue
// en el historial.
func (h *Handler) Exists(ctx context.Context, name string) (bool, error) {
	_, err := h.reader.GetByName(ctx, strings.TrimSpace(name))
	if errors.Is(err, domain.ErrAgentNotFound) {
		return false, nil
	}
	return err == nil, err
}
