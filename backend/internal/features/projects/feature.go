// Package projects construye las slices de la feature y registra sus entradas.
package projects

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/commands/addmember"
	"pmt/internal/features/projects/commands/adoptorphans"
	"pmt/internal/features/projects/commands/changeproductversion"
	"pmt/internal/features/projects/commands/changestatus"
	"pmt/internal/features/projects/commands/create"
	"pmt/internal/features/projects/commands/delete"
	"pmt/internal/features/projects/commands/removemember"
	"pmt/internal/features/projects/commands/update"
	"pmt/internal/features/projects/commands/updatemember"
	"pmt/internal/features/projects/domain"
	"pmt/internal/features/projects/queries/checkowner"
	"pmt/internal/features/projects/queries/checkplanning"
	"pmt/internal/features/projects/queries/checkstatus"
	"pmt/internal/features/projects/queries/checkworker"
	"pmt/internal/features/projects/queries/checkworking"
	"pmt/internal/features/projects/queries/get"
	"pmt/internal/features/projects/queries/holdof"
	"pmt/internal/features/projects/queries/list"
	"pmt/internal/features/projects/queries/listmembers"
	"pmt/internal/features/projects/queries/listproductversionchanges"
	"pmt/internal/features/projects/queries/liststatuschanges"
	"pmt/internal/features/projects/queries/roleof"
	"pmt/internal/features/projects/store"
	"pmt/internal/features/projects/store/sqlcgen"
	"pmt/internal/kernel"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
// stages y users los aportan esas features, conectadas en app.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB, stages changestatus.StageProgress, users domain.Users) {
	repo := store.NewProjects(db)
	statusChanges := store.NewStatusChanges(db)
	versionChanges := store.NewProductVersionChanges(db)

	createProject := create.New(repo)
	createProject.Register(mux)
	createProject.RegisterTool(tools)

	changeStatus := changestatus.New(struct {
		*store.Projects
		*store.StatusChanges
	}{repo, statusChanges}, stages)
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	updateProject := update.New(repo)
	updateProject.Register(mux)
	updateProject.RegisterTool(tools)

	changeProductVersion := changeproductversion.New(struct {
		*store.Projects
		*store.ProductVersionChanges
	}{repo, versionChanges})
	changeProductVersion.Register(mux)
	changeProductVersion.RegisterTool(tools)

	deleteProject := delete.New(repo)
	deleteProject.Register(mux)
	deleteProject.RegisterTool(tools)

	members := store.NewMembers(db)

	get.New(struct {
		*store.Projects
		*store.Members
	}{repo, members}).Register(mux)

	listProjects := list.New(repo)
	listProjects.Register(mux)
	listProjects.RegisterTool(tools)

	listStatusChanges := liststatuschanges.New(statusChanges)
	listStatusChanges.Register(mux)
	listStatusChanges.RegisterTool(tools)

	listVersionChanges := listproductversionchanges.New(versionChanges)
	listVersionChanges.Register(mux)
	listVersionChanges.RegisterTool(tools)

	listMembers := listmembers.New(members, users)
	listMembers.Register(mux)
	listMembers.RegisterTool(tools)

	addMember := addmember.New(members, users)
	addMember.Register(mux)
	addMember.RegisterTool(tools)

	updateMember := updatemember.New(members, users)
	updateMember.Register(mux)
	updateMember.RegisterTool(tools)

	removeMember := removemember.New(members, users)
	removeMember.Register(mux)
	removeMember.RegisterTool(tools)
}

// Membership resuelve a qué proyecto pertenece un elemento y qué rol tiene en él
// un usuario. La usan la comprobación de membresía de app y los eventos.
type Membership struct {
	roleOf *roleof.Handler
	holdOf *holdof.Handler
}

// NewMembership crea la consulta de membresía.
func NewMembership(db sqlcgen.DBTX) *Membership {
	members := store.NewMembers(db)
	return &Membership{roleOf: roleof.New(members), holdOf: holdof.New(members)}
}

// HoldOf devuelve qué contiene al elemento, stage o bug (vacío si no pertenece a
// ninguno, como un proyecto), y quién lo tiene tomado. Lo usa app para rechazar
// las escrituras sobre una etapa o un bug que tiene otra persona.
func (m *Membership) HoldOf(ctx context.Context, kind string, id uuid.UUID) (string, kernel.Hold, error) {
	result, err := m.holdOf.Handle(ctx, kind, id)
	return result.HolderKind, result.Hold, err
}

// RoleOf devuelve el rol del usuario en el proyecto del elemento, vacío si no es
// miembro. found es false si el elemento no existe. kind es project, stage, task,
// bug, finding, test_plan o dev_plan.
func (m *Membership) RoleOf(ctx context.Context, userID uuid.UUID, kind string, id uuid.UUID) (domain.Role, bool, error) {
	result, err := m.roleOf.Handle(ctx, userID, kind, id)
	return result.Role, result.Found, err
}

// Workers permite a otras features saber si un nombre es un usuario que puede
// trabajar en un proyecto (ser su TL, dev o dueño de una tarea).
type Workers struct {
	check *checkworker.Handler
}

// NewWorkers crea el verificador. users lo aporta la feature auth.
func NewWorkers(db sqlcgen.DBTX, users checkworker.Users) *Workers {
	return &Workers{check: checkworker.New(store.NewMembers(db), users)}
}

// Worker indica si el nombre es un usuario activo. Si lo es y no es owner ni
// editor del proyecto, responde conflict. Si no es un usuario activo devuelve
// false: quien pregunta lo valida como agente.
func (w *Workers) Worker(ctx context.Context, projectID uuid.UUID, username string) (bool, error) {
	return w.check.Handle(ctx, projectID, username)
}

// Owners permite a otras features saber si un usuario es el owner de un
// proyecto: solo él libera una etapa o un bug tomados.
type Owners struct {
	check *checkowner.Handler
}

// NewOwners crea el verificador.
func NewOwners(db sqlcgen.DBTX) *Owners {
	return &Owners{check: checkowner.New(store.NewMembers(db))}
}

// IsOwner indica si el usuario es el owner del proyecto.
func (o *Owners) IsOwner(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	return o.check.Handle(ctx, projectID, userID)
}

// AdoptOrphans deja al usuario como owner de los proyectos sin owner y devuelve
// cuántos adoptó. Se llama al arrancar, con el dueño de la instancia.
func AdoptOrphans(ctx context.Context, db sqlcgen.DBTX, owner uuid.UUID) (int64, error) {
	return adoptorphans.New(store.NewMembers(db)).Handle(ctx, owner)
}

// Checker permite a otras features saber qué admite un proyecto según su status.
type Checker struct {
	checkPlanning *checkplanning.Handler
	checkWorking  *checkworking.Handler
	checkStatus   *checkstatus.Handler
	get           *get.Handler
}

// NewChecker crea el verificador sobre las queries checkplanning, checkworking, checkstatus y get.
func NewChecker(db sqlcgen.DBTX) *Checker {
	repo := store.NewProjects(db)
	reader := struct {
		*store.Projects
		*store.Members
	}{repo, store.NewMembers(db)}
	return &Checker{checkPlanning: checkplanning.New(repo), checkWorking: checkworking.New(repo), checkStatus: checkstatus.New(repo), get: get.New(reader)}
}

// Check devuelve un error de negocio si el proyecto no existe.
func (c *Checker) Check(ctx context.Context, id uuid.UUID) error {
	_, err := c.get.Handle(ctx, kernel.Actor{}, id)
	return err
}

// EnsurePlanning devuelve un error de negocio si el proyecto no existe o no está en planning.
func (c *Checker) EnsurePlanning(ctx context.Context, id uuid.UUID) error {
	return c.checkPlanning.Handle(ctx, id)
}

// IsWorking indica si el proyecto está en working. Si no existe, devuelve un error de negocio.
func (c *Checker) IsWorking(ctx context.Context, id uuid.UUID) (bool, error) {
	return c.checkWorking.Handle(ctx, id)
}

// AcceptsWork indica si las etapas y tareas del proyecto pueden avanzar.
func (c *Checker) AcceptsWork(ctx context.Context, id uuid.UUID) (bool, error) {
	return c.checkStatus.AcceptsWork(ctx, id)
}

// IsFrozen indica si el proyecto congela sus etapas y tareas.
func (c *Checker) IsFrozen(ctx context.Context, id uuid.UUID) (bool, error) {
	return c.checkStatus.IsFrozen(ctx, id)
}

// EnsureWorking devuelve un error de negocio si el proyecto no existe o no está en working.
func (c *Checker) EnsureWorking(ctx context.Context, id uuid.UUID) error {
	working, err := c.checkWorking.Handle(ctx, id)
	if err != nil {
		return err
	}
	if !working {
		return domain.ErrProjectNotWorking
	}
	return nil
}
