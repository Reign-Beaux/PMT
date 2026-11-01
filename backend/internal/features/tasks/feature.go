// Package tasks construye las slices de la feature y registra sus entradas.
package tasks

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/tasks/commands/add"
	"pmt/internal/features/tasks/commands/assign"
	"pmt/internal/features/tasks/commands/changestatus"
	"pmt/internal/features/tasks/commands/changestatusbatch"
	"pmt/internal/features/tasks/commands/create"
	"pmt/internal/features/tasks/commands/delete"
	"pmt/internal/features/tasks/commands/retire"
	"pmt/internal/features/tasks/commands/revise"
	"pmt/internal/features/tasks/commands/update"
	"pmt/internal/features/tasks/queries/checkassigned"
	"pmt/internal/features/tasks/queries/checkdone"
	"pmt/internal/features/tasks/queries/checkrelease"
	"pmt/internal/features/tasks/queries/get"
	"pmt/internal/features/tasks/queries/hasactive"
	"pmt/internal/features/tasks/queries/list"
	"pmt/internal/features/tasks/queries/listchanges"
	"pmt/internal/features/tasks/queries/liststatuschanges"
	"pmt/internal/features/tasks/store"
	"pmt/internal/features/tasks/store/sqlcgen"
)

// StageChecker reúne lo que las slices piden a la feature stages.
type StageChecker interface {
	add.StageChecker
	create.StageChecker
	list.StageChecker
	changestatus.StageChecker
	assign.StageChecker
	changestatusbatch.StageChecker
}

// AgentChecker reúne lo que las slices piden a la feature agents.
type AgentChecker interface {
	assign.AgentChecker
}

// Register registra las rutas REST y las herramientas MCP de la feature.
// stages y agents los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, stages StageChecker, agents AgentChecker) {
	repo := store.NewTasks(db)
	changes := store.NewChanges(db)
	statusChanges := store.NewStatusChanges(db)

	createTask := create.New(repo, stages)
	createTask.Register(mux)
	createTask.RegisterTool(tools)

	updateTask := update.New(repo, stages)
	updateTask.Register(mux)
	updateTask.RegisterTool(tools)

	deleteTask := delete.New(repo, stages)
	deleteTask.Register(mux)
	deleteTask.RegisterTool(tools)

	changeStatus := changestatus.New(struct {
		*store.Tasks
		*store.StatusChanges
	}{repo, statusChanges}, stages)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	changeBatch := changestatusbatch.New(statusChanges, stages)
	changeBatch.Register(mux)
	changeBatch.RegisterTool(tools)

	assignTask := assign.New(repo, stages, agents)
	assignTask.Register(mux)
	assignTask.RegisterTool(tools)

	get.New(repo).Register(mux)

	listTasks := list.New(repo, stages)
	listTasks.Register(mux)
	listTasks.RegisterTool(tools)

	addTask := add.New(changes, stages)
	addTask.Register(mux)
	addTask.RegisterTool(tools)

	reviseTask := revise.New(struct {
		*store.Tasks
		*store.Changes
	}{repo, changes}, stages)
	reviseTask.Register(mux)
	reviseTask.RegisterTool(tools)

	retireTask := retire.New(struct {
		*store.Tasks
		*store.Changes
	}{repo, changes}, stages)
	retireTask.Register(mux)
	retireTask.RegisterTool(tools)

	listStatusChanges := liststatuschanges.New(statusChanges)
	listStatusChanges.Register(mux)
	listStatusChanges.RegisterTool(tools)

	listChanges := listchanges.New(changes)
	listChanges.Register(mux)
	listChanges.RegisterTool(tools)
}

// Checker permite a otras features saber si las tareas de una etapa están
// terminadas, si tiene tareas sin retirar y si tiene tareas activas.
type Checker struct {
	checkDone *checkdone.Handler
	hasActive *hasactive.Handler
	assigned  *checkassigned.Handler
	release   *checkrelease.Handler
}

// NewChecker crea el verificador sobre las queries checkdone y hasactive.
func NewChecker(db sqlcgen.DBTX) *Checker {
	repo := store.NewTasks(db)
	return &Checker{checkDone: checkdone.New(repo), hasActive: hasactive.New(repo), assigned: checkassigned.New(repo), release: checkrelease.New(repo)}
}

// EnsureClosable devuelve un error de negocio si las tareas de la etapa no permiten cerrarla.
func (c *Checker) EnsureClosable(ctx context.Context, stageID uuid.UUID) error {
	closure, err := c.checkDone.Handle(ctx, stageID)
	if err != nil {
		return err
	}
	return closure.EnsureClosable()
}

// EnsureReleasable devuelve un error de negocio si la etapa no tiene tareas activas.
func (c *Checker) EnsureReleasable(ctx context.Context, stageID uuid.UUID) error {
	return c.release.Handle(ctx, stageID)
}

// HasOpenAssigned indica si el agente tiene tareas sin terminar en la etapa.
func (c *Checker) HasOpenAssigned(ctx context.Context, stageID uuid.UUID, agent string) (bool, error) {
	return c.assigned.Handle(ctx, stageID, agent)
}

// HasActive indica si la etapa tiene tareas sin retirar.
func (c *Checker) HasActive(ctx context.Context, stageID uuid.UUID) (bool, error) {
	return c.hasActive.Handle(ctx, stageID)
}
