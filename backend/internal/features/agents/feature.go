// Package agents construye las slices de la feature. No tiene REST ni MCP: los
// agentes y sus tokens solo se administran desde la línea de comandos, para que
// ningún agente pueda emitirlos.
package agents

import (
	"context"

	"pmt/internal/features/agents/commands/create"
	"pmt/internal/features/agents/commands/revoke"
	"pmt/internal/features/agents/commands/rotate"
	"pmt/internal/features/agents/queries/authenticate"
	"pmt/internal/features/agents/queries/list"
	"pmt/internal/features/agents/queries/lookup"
	"pmt/internal/features/agents/store"
	"pmt/internal/features/agents/store/sqlcgen"
	"pmt/internal/kernel"
)

// Admin reúne las operaciones de administración.
type Admin struct {
	Create *create.Handler
	Rotate *rotate.Handler
	Revoke *revoke.Handler
	List   *list.Handler
}

// NewAdmin construye las operaciones de administración. users es la feature
// auth: un agente no puede llamarse como un usuario.
func NewAdmin(db sqlcgen.DBTX, users create.Users) Admin {
	repo := store.NewAgents(db)
	return Admin{
		Create: create.New(repo, users),
		Rotate: rotate.New(repo),
		Revoke: revoke.New(repo),
		List:   list.New(repo),
	}
}

// NewAuthenticator crea el autenticador de tokens de agente.
func NewAuthenticator(db sqlcgen.DBTX) *authenticate.Handler {
	return authenticate.New(store.NewAgents(db))
}

// Checker permite a otras features conocer el perfil de un agente activo.
type Checker struct {
	lookup *lookup.Handler
}

// NewChecker crea el verificador sobre la query lookup.
func NewChecker(db sqlcgen.DBTX) *Checker {
	return &Checker{lookup: lookup.New(store.NewAgents(db))}
}

// Exists indica si hay un agente con ese nombre, revocado o no.
func (c *Checker) Exists(ctx context.Context, name string) (bool, error) {
	return c.lookup.Exists(ctx, name)
}

// Profile devuelve el perfil del agente, o un error de negocio si no existe o está revocado.
func (c *Checker) Profile(ctx context.Context, name string) (kernel.Profile, error) {
	return c.lookup.Handle(ctx, name)
}
