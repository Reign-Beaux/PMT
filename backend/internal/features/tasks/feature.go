// Package tasks construye las slices de la feature y registra sus entradas.
package tasks

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/commands/create"
	"pmt/internal/features/tasks/commands/delete"
	"pmt/internal/features/tasks/commands/update"
	"pmt/internal/features/tasks/queries/checkdone"
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

// Checker permite a otras features saber si las tareas de una etapa están terminadas.
type Checker struct {
	checkDone *checkdone.Handler
}

// NewChecker crea el verificador sobre la query checkdone.
func NewChecker(db sqlcgen.DBTX) *Checker {
	return &Checker{checkDone: checkdone.New(store.NewTasks(db))}
}

// AllDone indica si todas las tareas de la etapa están en done.
func (c *Checker) AllDone(ctx context.Context, stageID uuid.UUID) (bool, error) {
	return c.checkDone.Handle(ctx, stageID)
}
