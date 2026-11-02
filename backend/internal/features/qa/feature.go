// Package qa construye las slices de la feature y registra sus entradas.
package qa

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/qa/commands/approve"
	"pmt/internal/features/qa/commands/create"
	"pmt/internal/features/qa/commands/returnplan"
	"pmt/internal/features/qa/commands/revise"
	"pmt/internal/features/qa/commands/submit"
	"pmt/internal/features/qa/queries/list"
	"pmt/internal/features/qa/queries/listversions"
	"pmt/internal/features/qa/store"
)

// StageChecker reúne lo que las slices piden a stages, conectado en app.
type StageChecker interface {
	create.StageChecker
	approve.StageChecker
}

// Register registra las rutas REST y las herramientas MCP de la feature. tasks y
// stages los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, tasks create.TaskChecker, stages StageChecker) {
	repo := store.NewPlans(db)

	createPlan := create.New(repo, tasks, stages)
	createPlan.Register(mux)
	createPlan.RegisterTool(tools)

	revisePlan := revise.New(repo, tasks)
	revisePlan.Register(mux)
	revisePlan.RegisterTool(tools)

	submitPlan := submit.New(repo)
	submitPlan.Register(mux)
	submitPlan.RegisterTool(tools)

	approvePlan := approve.New(repo, stages)
	approvePlan.Register(mux)
	approvePlan.RegisterTool(tools)

	returnPlan := returnplan.New(repo, stages)
	returnPlan.Register(mux)
	returnPlan.RegisterTool(tools)

	listPlans := list.New(repo)
	listPlans.Register(mux)
	listPlans.RegisterTool(tools)

	listVersions := listversions.New(repo)
	listVersions.Register(mux)
	listVersions.RegisterTool(tools)
}
