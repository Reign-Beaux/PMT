// Package stages construye las slices de la feature y registra sus entradas.
package stages

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/stages/commands/add"
	"pmt/internal/features/stages/commands/addmember"
	"pmt/internal/features/stages/commands/assignlead"
	"pmt/internal/features/stages/commands/changestatus"
	"pmt/internal/features/stages/commands/create"
	"pmt/internal/features/stages/commands/delete"
	"pmt/internal/features/stages/commands/removemember"
	"pmt/internal/features/stages/commands/retire"
	"pmt/internal/features/stages/commands/revise"
	"pmt/internal/features/stages/commands/update"
	"pmt/internal/features/stages/domain"
	"pmt/internal/features/stages/queries/checkprogress"
	"pmt/internal/features/stages/queries/checktasks"
	"pmt/internal/features/stages/queries/checkteam"
	"pmt/internal/features/stages/queries/get"
	"pmt/internal/features/stages/queries/list"
	"pmt/internal/features/stages/queries/listchanges"
	"pmt/internal/features/stages/queries/liststatuschanges"
	"pmt/internal/features/stages/store"
	"pmt/internal/features/stages/store/sqlcgen"
	"pmt/internal/kernel"
)

// TaskChecker reúne lo que las slices piden a la feature tasks.
type TaskChecker interface {
	changestatus.TaskChecker
	retire.TaskChecker
	removemember.TaskChecker
}

// ProjectChecker reúne lo que las slices piden a la feature projects.
type ProjectChecker interface {
	add.ProjectChecker
	update.ProjectChecker
	checktasks.ProjectChecker
}

// AgentChecker reúne lo que las slices piden a la feature agents.
type AgentChecker interface {
	assignlead.AgentChecker
	addmember.AgentChecker
}

// StageCloser cierra, cancela o retira una etapa y mueve el proyecto si termina la
// versión 1. Lo implementa la feature workflow.
type StageCloser interface {
	CloseStage(ctx context.Context, s domain.Stage, change domain.StatusChange, actor kernel.Actor) (domain.Stage, error)
	RetireStage(ctx context.Context, s domain.Stage, change domain.Change, actor kernel.Actor) (domain.Change, error)
}

// Register registra las rutas REST y las herramientas MCP de la feature.
// tasks, projects, agents y closer los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, tasks TaskChecker, projects ProjectChecker, agents AgentChecker, closer StageCloser) {
	repo := store.NewStages(db)
	changes := store.NewChanges(db)
	statusChanges := store.NewStatusChanges(db)

	createStage := create.New(repo, projects)
	createStage.Register(mux)
	createStage.RegisterTool(tools)

	updateStage := update.New(repo, projects)
	updateStage.Register(mux)
	updateStage.RegisterTool(tools)

	changeStatus := changestatus.New(struct {
		*store.Stages
		*store.Changes
		*store.StatusChanges
		StageCloser
	}{repo, changes, statusChanges, closer}, tasks, projects)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	assignLead := assignlead.New(repo, agents)
	assignLead.Register(mux)
	assignLead.RegisterTool(tools)

	addMember := addmember.New(repo, agents)
	addMember.Register(mux)
	addMember.RegisterTool(tools)

	removeMember := removemember.New(repo, tasks)
	removeMember.Register(mux)
	removeMember.RegisterTool(tools)

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
		StageCloser
	}{repo, closer}, tasks)
	retireStage.Register(mux)
	retireStage.RegisterTool(tools)

	listStatusChanges := liststatuschanges.New(statusChanges)
	listStatusChanges.Register(mux)
	listStatusChanges.RegisterTool(tools)

	listChanges := listchanges.New(changes)
	listChanges.Register(mux)
	listChanges.RegisterTool(tools)
}

// Checker permite a otras features confirmar que una etapa existe y qué
// operaciones admiten sus tareas.
type Checker struct {
	get   *get.Handler
	tasks *checktasks.Handler
	team  *checkteam.Handler
}

// NewChecker crea el verificador sobre las queries get y checktasks.
func NewChecker(db sqlcgen.DBTX, projects ProjectChecker) *Checker {
	repo := store.NewStages(db)
	return &Checker{get: get.New(repo), tasks: checktasks.New(repo, projects), team: checkteam.New(repo)}
}

// Team devuelve el TL y los devs de la etapa.
func (c *Checker) Team(ctx context.Context, id uuid.UUID) (kernel.Team, error) {
	return c.team.Handle(ctx, id)
}

// Check devuelve un error de negocio si la etapa no existe.
func (c *Checker) Check(ctx context.Context, id uuid.UUID) error {
	_, err := c.get.Handle(ctx, id)
	return err
}

// EnsureWorkOpen devuelve un error de negocio si una tarea de la etapa no puede avanzar.
func (c *Checker) EnsureWorkOpen(ctx context.Context, id uuid.UUID) error {
	return c.tasks.EnsureWorkOpen(ctx, id)
}

// EnsureTaskCancellable devuelve un error de negocio si una tarea de la etapa no
// se puede cancelar ni reabrir.
func (c *Checker) EnsureTaskCancellable(ctx context.Context, id uuid.UUID) error {
	return c.tasks.EnsureTaskCancellable(ctx, id)
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

// NewProgress crea el verificador que usa projects para saber si sus etapas
// permiten un cambio de status. No depende de projects, así que app lo construye
// antes de registrar esa feature.
func NewProgress(db sqlcgen.DBTX) *checkprogress.Handler {
	return checkprogress.New(store.NewStages(db))
}
