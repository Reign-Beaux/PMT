// Package bugs construye las slices de la feature y registra sus entradas.
package bugs

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/bugs/commands/changestatus"
	"pmt/internal/features/bugs/commands/create"
	"pmt/internal/features/bugs/commands/update"
	"pmt/internal/features/bugs/queries/list"
	"pmt/internal/features/bugs/queries/liststatuschanges"
	"pmt/internal/features/bugs/store"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
// projects lo aporta la feature projects, conectada en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, projects create.ProjectChecker) {
	repo := store.NewBugs(db)
	statusChanges := store.NewStatusChanges(db)

	createBug := create.New(repo, projects)
	createBug.Register(mux)
	createBug.RegisterTool(tools)

	updateBug := update.New(repo, projects)
	updateBug.Register(mux)
	updateBug.RegisterTool(tools)

	changeStatus := changestatus.New(struct {
		*store.Bugs
		*store.StatusChanges
	}{repo, statusChanges})
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	listBugs := list.New(repo)
	listBugs.Register(mux)
	listBugs.RegisterTool(tools)

	listStatusChanges := liststatuschanges.New(statusChanges)
	listStatusChanges.Register(mux)
	listStatusChanges.RegisterTool(tools)
}
