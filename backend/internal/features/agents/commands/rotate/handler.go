// Package rotate sustituye el token de un agente.
package rotate

import (
	"context"

	"github.com/google/uuid"

	"pmt/internal/features/agents/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	GetByName(ctx context.Context, name string) (domain.Agent, error)
	UpdateTokenHash(ctx context.Context, id uuid.UUID, tokenHash []byte) error
}

// Handler ejecuta la operación.
type Handler struct {
	repo Repository
}

// New crea el handler.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Handle genera un token nuevo y devuelve su valor. El anterior deja de valer.
func (h *Handler) Handle(ctx context.Context, name string) (string, error) {
	agent, err := h.repo.GetByName(ctx, name)
	if err != nil {
		return "", err
	}
	if agent.Revoked() {
		return "", domain.ErrAgentRevoked
	}

	token := domain.NewToken()
	if err := h.repo.UpdateTokenHash(ctx, agent.ID, domain.HashToken(token)); err != nil {
		return "", err
	}
	return token, nil
}
