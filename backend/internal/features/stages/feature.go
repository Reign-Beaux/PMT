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

// Register registra las rutas REST y las herramientas MCP de la feature.
// tasks y projects los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, tasks TaskChecker, projects update.ProjectChecker) {
	repo := store.NewStages(db)
	changes := store.NewChanges(db)

	createStage := create.New(repo, projects)
	createStage.Register(mux)
	createStage.RegisterTool(tools)

	updateStage := update.New(repo, projects)
	updateStage.Register(mux)
	updateStage.RegisterTool(tools)

	changeStatus := changestatus.New(repo, tasks)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	deleteStage := delete.New(repo, projects)
	deleteStage.Register(mux)
	deleteStage.RegisterTool(tools)

	get.New(repo).Register(mux)

	listStages := list.New(repo)
	listStages.Register(mux)
	listStages.RegisterTool(tools)

	addStage := add.New(changes)
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

// Checker permite a otras features confirmar que una etapa existe y que su
// proyecto está en planning.
type Checker struct {
	get      *get.Handler
	projects update.ProjectChecker
}

// NewChecker crea el verificador sobre la query get.
func NewChecker(db sqlcgen.DBTX, projects update.ProjectChecker) *Checker {
	return &Checker{get: get.New(store.NewStages(db)), projects: projects}
}

// Check devuelve un error de negocio si la etapa no existe.
func (c *Checker) Check(ctx context.Context, id uuid.UUID) error {
	_, err := c.get.Handle(ctx, id)
	return err
}

// EnsureProjectPlanning devuelve un error de negocio si la etapa no existe o su
// proyecto no está en planning.
func (c *Checker) EnsureProjectPlanning(ctx context.Context, id uuid.UUID) error {
	stage, err := c.get.Handle(ctx, id)
	if err != nil {
		return err
	}
	return c.projects.EnsurePlanning(ctx, stage.ProjectID)
}
