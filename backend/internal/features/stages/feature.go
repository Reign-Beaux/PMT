// Package stages construye las slices de la feature y registra sus entradas.
package stages

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/commands/changestatus"
	"pmt/internal/features/stages/commands/create"
	"pmt/internal/features/stages/commands/delete"
	"pmt/internal/features/stages/commands/update"
	"pmt/internal/features/stages/queries/get"
	"pmt/internal/features/stages/queries/list"
	"pmt/internal/features/stages/store"
	"pmt/internal/features/stages/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
// tasks lo aporta la feature tasks, conectada en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX, tasks changestatus.TaskChecker) {
	repo := store.NewStages(db)

	createStage := create.New(repo)
	createStage.Register(mux)
	createStage.RegisterTool(tools)

	update.New(repo).Register(mux)

	changeStatus := changestatus.New(repo, tasks)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	delete.New(repo).Register(mux)
	get.New(repo).Register(mux)

	listStages := list.New(repo)
	listStages.Register(mux)
	listStages.RegisterTool(tools)
}

// Checker permite a otras features confirmar que una etapa existe.
type Checker struct {
	get *get.Handler
}

// NewChecker crea el verificador sobre la query get.
func NewChecker(db sqlcgen.DBTX) *Checker {
	return &Checker{get: get.New(store.NewStages(db))}
}

// Check devuelve un error de negocio si la etapa no existe.
func (c *Checker) Check(ctx context.Context, id uuid.UUID) error {
	_, err := c.get.Handle(ctx, id)
	return err
}
