// Package bugs construye las slices de la feature y registra sus entradas.
package bugs

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/bugs/commands/changestatus"
	"pmt/internal/features/bugs/commands/create"
	"pmt/internal/features/bugs/commands/release"
	"pmt/internal/features/bugs/commands/update"
	"pmt/internal/features/bugs/queries/get"
	"pmt/internal/features/bugs/queries/list"
	"pmt/internal/features/bugs/queries/liststatuschanges"
	"pmt/internal/features/bugs/store"
	"pmt/internal/features/bugs/store/sqlcgen"
	"pmt/internal/kernel"
)

// ProjectChecker reúne lo que las slices piden a la feature projects.
type ProjectChecker interface {
	create.ProjectChecker
	list.ProjectChecker
}

// Register registra las rutas REST y las herramientas MCP de la feature.
// projects y owners los aporta la feature projects, conectada en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, projects ProjectChecker, owners release.Owners) {
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
	}{repo, statusChanges}, projects)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	listBugs := list.New(repo, projects)
	listBugs.Register(mux)
	listBugs.RegisterTool(tools)

	listStatusChanges := liststatuschanges.New(statusChanges)
	listStatusChanges.Register(mux)
	listStatusChanges.RegisterTool(tools)

	releaseBug := release.New(struct {
		*store.Bugs
		*store.StatusChanges
	}{repo, statusChanges}, owners)
	releaseBug.Register(mux)
	releaseBug.RegisterTool(tools)
}

// AdoptHolds deja tomados por el usuario los bugs en corrección que nadie tiene y
// devuelve cuántos. Se llama al arrancar, con el dueño de la instancia, solo
// cuando la migración de bloqueos no encontró usuarios.
func AdoptHolds(ctx context.Context, db store.DB, owner kernel.UserRef) (int64, error) {
	return store.NewStatusChanges(db).AdoptHolds(ctx, owner)
}

// Checker permite a otras features confirmar que un bug existe.
type Checker struct {
	get *get.Handler
}

// NewChecker crea el verificador sobre la query get.
func NewChecker(db sqlcgen.DBTX) *Checker {
	return &Checker{get: get.New(store.NewBugs(db))}
}

// Check devuelve un error de negocio si el bug no existe.
func (c *Checker) Check(ctx context.Context, id uuid.UUID) error {
	return c.get.Handle(ctx, id)
}
