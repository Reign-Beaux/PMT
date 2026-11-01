// Package activity construye las slices de la feature y registra sus entradas.
package activity

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/activity/commands/add"
	"pmt/internal/features/activity/queries/list"
	"pmt/internal/features/activity/store"
	"pmt/internal/features/activity/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature. projects,
// stages, tasks y bugs los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX, projects add.ProjectChecker, stages add.StageChecker, tasks add.TaskChecker, bugs add.BugChecker) {
	repo := store.NewComments(db)

	addComment := add.New(repo, projects, stages, tasks, bugs)
	addComment.Register(mux)
	addComment.RegisterTool(tools)

	listComments := list.New(repo)
	listComments.Register(mux)
	listComments.RegisterTool(tools)
}
