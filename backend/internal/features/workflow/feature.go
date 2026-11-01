// Package workflow compone las operaciones que mueven varios agregados en una
// transacción: release_project, start_stage y el cierre y el retiro de una etapa.
package workflow

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/workflow/commands/releaseproject"
	"pmt/internal/features/workflow/commands/startstage"
	"pmt/internal/features/workflow/store"
)

// Register registra las rutas REST y las herramientas MCP de la feature. tasks lo
// aporta esa feature, conectada en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, tasks releaseproject.TaskChecker) {
	repo := store.NewWorkflow(db)

	release := releaseproject.New(repo, tasks)
	release.Register(mux)
	release.RegisterTool(tools)

	start := startstage.New(repo)
	start.Register(mux)
	start.RegisterTool(tools)
}

// NewCloser crea lo que stages usa para cerrar una etapa. stages no importa esta
// feature: app se lo pasa.
func NewCloser(db store.DB) *store.Workflow {
	return store.NewWorkflow(db)
}
