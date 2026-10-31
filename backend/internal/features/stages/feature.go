// Package stages construye las slices de la feature y registra sus entradas.
package stages

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/commands/add"
	"pmt/internal/features/stages/commands/changestatus"
	"pmt/internal/features/stages/commands/create"
	"pmt/internal/features/stages/commands/delete"
	"pmt/internal/features/stages/commands/retire"
	"pmt/internal/features/stages/commands/revise"
	"pmt/internal/features/stages/commands/update"
	"pmt/internal/features/stages/queries/checktasks"
	"pmt/internal/features/stages/queries/get"
	"pmt/internal/features/stages/queries/list"
	"pmt/internal/features/stages/queries/listchanges"
	"pmt/internal/features/stages/store"
	"pmt/internal/features/stages/store/sqlcgen"
)

// TaskChecker reúne lo que las slices piden a la feature tasks.
type TaskChecker interface {
	changestatus.TaskChecker
	retire.TaskChecker
}

// ProjectChecker reúne lo que las slices piden a la feature projects.
type ProjectChecker interface {
	add.ProjectChecker
	update.ProjectChecker
}

// Register registra las rutas REST y las herramientas MCP de la feature.
// tasks y projects los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, tasks TaskChecker, projects ProjectChecker) {
	repo := store.NewStages(db)
	changes := store.NewChanges(db)

	createStage := create.New(repo, projects)
	createStage.Register(mux)
	createStage.RegisterTool(tools)

	updateStage := update.New(repo, projects)
	updateStage.Register(mux)
	updateStage.RegisterTool(tools)

	changeStatus := changestatus.New(struct {
		*store.Stages
		*store.Changes
	}{repo, changes}, tasks)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	deleteStage := delete.New(repo, projects)
	deleteStage.Register(mux)
	deleteStage.RegisterTool(tools)

	get.New(repo).Register(mux)

	listStages := list.New(repo)
	listStages.Register(mux)
	listStages.RegisterTool(tools)

	addStage := add.New(struct {
		*store.Stages
		*store.Changes
	}{repo, changes}, projects)
	addStage.Register(mux)
	addStage.RegisterTool(tools)

	reviseStage := revise.New(struct {
		*store.Stages
		*store.Changes
	}{repo, changes})
	reviseStage.Register(mux)
	reviseStage.RegisterTool(tools)

	retireStage := retire.New(struct {
		*store.Stages
		*store.Changes
	}{repo, changes}, tasks)
	retireStage.Register(mux)
	retireStage.RegisterTool(tools)

	listChanges := listchanges.New(changes)
	listChanges.Register(mux)
	listChanges.RegisterTool(tools)
}

// Checker permite a otras features confirmar que una etapa existe y qué
// operaciones admiten sus tareas.
type Checker struct {
	get   *get.Handler
	tasks *checktasks.Handler
}

// NewChecker crea el verificador sobre las queries get y checktasks.
func NewChecker(db sqlcgen.DBTX, projects ProjectChecker) *Checker {
	repo := store.NewStages(db)
	return &Checker{get: get.New(repo), tasks: checktasks.New(repo, projects)}
}

// Check devuelve un error de negocio si la etapa no existe.
func (c *Checker) Check(ctx context.Context, id uuid.UUID) error {
	_, err := c.get.Handle(ctx, id)
	return err
}

// EnsureTasksEditable devuelve un error de negocio si la etapa no existe o sus
// tareas no se pueden crear, editar ni eliminar directamente.
func (c *Checker) EnsureTasksEditable(ctx context.Context, id uuid.UUID) error {
	return c.tasks.EnsureEditable(ctx, id)
}

// EnsureHistoryRule devuelve un error de negocio si la etapa no existe o está
// en su planning propio, donde sus tareas no admiten cambios con historial.
func (c *Checker) EnsureHistoryRule(ctx context.Context, id uuid.UUID) error {
	return c.tasks.EnsureHistoryRule(ctx, id)
}
