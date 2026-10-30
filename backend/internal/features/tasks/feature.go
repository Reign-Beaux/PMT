// Package tasks construye las slices de la feature y registra sus entradas.
package tasks

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/commands/create"
	"pmt/internal/features/tasks/commands/delete"
	"pmt/internal/features/tasks/commands/update"
	"pmt/internal/features/tasks/queries/get"
	"pmt/internal/features/tasks/queries/list"
	"pmt/internal/features/tasks/store"
	"pmt/internal/features/tasks/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
// stages lo aporta la feature stages, conectada en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX, stages list.StageChecker) {
	repo := store.NewTasks(db)

	createTask := create.New(repo)
	createTask.Register(mux)
	createTask.RegisterTool(tools)

	update.New(repo).Register(mux)
	delete.New(repo).Register(mux)
	get.New(repo).Register(mux)

	listTasks := list.New(repo, stages)
	listTasks.Register(mux)
	listTasks.RegisterTool(tools)
}
