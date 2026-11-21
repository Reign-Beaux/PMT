// Package create registra un agente y genera su token.
package create

import (
	"context"

	"pmt/internal/features/agents/domain"
	"pmt/internal/kernel"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	Create(ctx context.Context, agent domain.Agent, tokenHash []byte) (domain.Agent, error)
}

// Command contiene los datos de entrada.
type Command struct {
	Name    string
	Profile kernel.Profile
}

// Result es el agente con su token, que no se puede volver a obtener.
type Result struct {
	Name    string
	Profile kernel.Profile
	Token   string
}

// Users indica si un nombre ya es el username de un usuario. Lo cumple la feature auth.
type Users interface {
	Exists(ctx context.Context, username string) (bool, error)
}

// Handler ejecuta la operación.
type Handler struct {
	repo  Repository
	users Users
}

// New crea el handler.
func New(repo Repository, users Users) *Handler {
	return &Handler{repo: repo, users: users}
}

// Handle crea el agente. Solo guarda el hash del token. Un nombre que ya es
// username de un usuario se rechaza: el historial no podría distinguirlos.
func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	agent, err := domain.NewAgent(cmd.Name, cmd.Profile)
	if err != nil {
		return Result{}, err
	}
	taken, err := h.users.Exists(ctx, agent.Name)
	if err != nil {
		return Result{}, err
	}
	if taken {
		return Result{}, domain.ErrAgentNameIsUser
	}

	token := domain.NewToken()
	created, err := h.repo.Create(ctx, agent, domain.HashToken(token))
	if err != nil {
		return Result{}, err
	}
	return Result{Name: created.Name, Profile: created.Profile, Token: token}, nil
}
