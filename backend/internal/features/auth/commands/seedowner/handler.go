// Package seedowner crea al dueño de la instancia al arrancar, si no hay usuarios.
package seedowner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"pmt/internal/features/auth/domain"
)

// Repository es la persistencia que necesita la operación.
type Repository interface {
	HasUsers(ctx context.Context) (bool, error)
	CreateOwner(ctx context.Context, user domain.User, passwordHash string) (bool, error)
}

// Command es la configuración owner.
type Command struct {
	Email    string
	Password string
	Username string
}

// Handler ejecuta la operación.
type Handler struct {
	repo   Repository
	agents domain.AgentNames
}

// New crea el handler.
func New(repo Repository, agents domain.AgentNames) *Handler {
	return &Handler{repo: repo, agents: agents}
}

// Handle crea al dueño con su username como nombre. Si ya hay usuarios no hace
// nada. Sin correo ni contraseña tampoco, y avisa si solo falta uno.
func (h *Handler) Handle(ctx context.Context, cmd Command) error {
	if cmd.Email == "" || cmd.Password == "" {
		if cmd.Email != "" || cmd.Password != "" {
			slog.Warn("configuración incompleta del dueño: falta owner.email u owner.password")
		}
		return nil
	}
	exists, err := h.repo.HasUsers(ctx)
	if err != nil || exists {
		return err
	}
	email, err := domain.ValidateEmail(cmd.Email)
	if err != nil {
		return fmt.Errorf("owner.email: %w", err)
	}
	username, err := domain.ValidateUsername(cmd.Username)
	if err != nil {
		return fmt.Errorf("owner.username: %w", err)
	}
	if err = domain.ValidatePassword(cmd.Password); err != nil {
		return fmt.Errorf("owner.password: %w", err)
	}
	if err = domain.EnsureUsernameFree(ctx, h.agents, username); err != nil {
		return fmt.Errorf("owner.username: %w", err)
	}
	hash, err := domain.HashPassword(cmd.Password)
	if err != nil {
		return err
	}
	owner := domain.User{ID: uuid.New(), Email: email, Username: username, Name: username, IsInstanceOwner: true}
	created, err := h.repo.CreateOwner(ctx, owner, hash)
	if err != nil {
		return err
	}
	if created {
		slog.Info("dueño de la instancia creado", "username", username)
	}
	return nil
}
