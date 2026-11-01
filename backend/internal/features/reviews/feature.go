// Package reviews construye las slices de la feature y registra sus entradas.
package reviews

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	closefinding "pmt/internal/features/reviews/commands/close"
	"pmt/internal/features/reviews/commands/create"
	"pmt/internal/features/reviews/queries/list"
	"pmt/internal/features/reviews/store"
	"pmt/internal/features/reviews/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature. tasks,
// bugs y stages los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX, tasks create.TaskChecker, bugs create.BugChecker, stages closefinding.StageChecker) {
	repo := store.NewFindings(db)

	createFinding := create.New(repo, tasks, bugs)
	createFinding.Register(mux)
	createFinding.RegisterTool(tools)

	closeFinding := closefinding.New(repo, tasks, stages)
	closeFinding.Register(mux)
	closeFinding.RegisterTool(tools)

	listFindings := list.New(repo)
	listFindings.Register(mux)
	listFindings.RegisterTool(tools)
}
