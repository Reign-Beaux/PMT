// Package agents construye las slices de la feature. No tiene REST ni MCP: los
// agentes y sus tokens solo se administran desde la línea de comandos, para que
// ningún agente pueda emitirlos.
package agents

import (
	"pmt/internal/features/agents/commands/create"
	"pmt/internal/features/agents/commands/revoke"
	"pmt/internal/features/agents/commands/rotate"
	"pmt/internal/features/agents/queries/authenticate"
	"pmt/internal/features/agents/queries/list"
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

// NewAdmin construye las operaciones de administración.
func NewAdmin(db sqlcgen.DBTX) Admin {
	repo := store.NewAgents(db)
	return Admin{
		Create: create.New(repo),
		Rotate: rotate.New(repo),
		Revoke: revoke.New(repo),
		List:   list.New(repo),
	}
}

// NewAuthenticator crea el autenticador que usan REST y MCP.
func NewAuthenticator(db sqlcgen.DBTX) kernel.Authenticator {
	return authenticate.New(store.NewAgents(db))
}
