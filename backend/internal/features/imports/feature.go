// Package imports construye las slices de la feature y registra sus entradas.
package imports

import (
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/imports/commands/importproject"
	"pmt/internal/features/imports/commands/importstage"
	"pmt/internal/features/imports/store"
)

// Register registra las herramientas MCP de la feature. No tiene rutas REST.
func Register(tools *sdk.Server, db store.Beginner) {
	repo := store.NewImports(db)
	importproject.New(repo).RegisterTool(tools)
	importstage.New(repo).RegisterTool(tools)
}
