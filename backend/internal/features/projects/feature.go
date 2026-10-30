// Package projects construye las slices de la feature y registra sus entradas.
package projects

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/commands/create"
	"pmt/internal/features/projects/commands/delete"
	"pmt/internal/features/projects/commands/update"
	"pmt/internal/features/projects/queries/get"
	"pmt/internal/features/projects/queries/list"
	"pmt/internal/features/projects/store"
	"pmt/internal/features/projects/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX) {
	repo := store.NewProjects(db)

	createProject := create.New(repo)
	createProject.Register(mux)
	createProject.RegisterTool(tools)

	update.New(repo).Register(mux)
	delete.New(repo).Register(mux)
	get.New(repo).Register(mux)
	list.New(repo).Register(mux)
}
