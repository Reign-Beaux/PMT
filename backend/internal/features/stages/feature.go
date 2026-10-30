// Package stages construye las slices de la feature y registra sus entradas.
package stages

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/commands/create"
	"pmt/internal/features/stages/commands/delete"
	"pmt/internal/features/stages/commands/update"
	"pmt/internal/features/stages/queries/get"
	"pmt/internal/features/stages/queries/list"
	"pmt/internal/features/stages/store"
	"pmt/internal/features/stages/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX) {
	repo := store.NewStages(db)

	createStage := create.New(repo)
	createStage.Register(mux)
	createStage.RegisterTool(tools)

	update.New(repo).Register(mux)
	delete.New(repo).Register(mux)
	get.New(repo).Register(mux)
	list.New(repo).Register(mux)
}
