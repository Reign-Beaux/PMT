// Package devplans construye las slices de la feature y registra sus entradas.
package devplans

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/devplans/commands/create"
	"pmt/internal/features/devplans/commands/revise"
	"pmt/internal/features/devplans/queries/list"
	"pmt/internal/features/devplans/queries/listversions"
	"pmt/internal/features/devplans/store"
)

// Register registra las rutas REST y las herramientas MCP de la feature. tasks,
// stages y bugs los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, tasks create.TaskChecker, stages, bugs create.Checker) {
	repo := store.NewPlans(db)

	createPlan := create.New(repo, tasks, stages, bugs)
	createPlan.Register(mux)
	createPlan.RegisterTool(tools)

	revisePlan := revise.New(repo, tasks, stages, bugs)
	revisePlan.Register(mux)
	revisePlan.RegisterTool(tools)

	listPlans := list.New(repo)
	listPlans.Register(mux)
	listPlans.RegisterTool(tools)

	listVersions := listversions.New(repo)
	listVersions.Register(mux)
	listVersions.RegisterTool(tools)
}
