// Package projects construye las slices de la feature y registra sus entradas.
package projects

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/features/projects/commands/changestatus"
	"pmt/internal/features/projects/commands/create"
	"pmt/internal/features/projects/commands/delete"
	"pmt/internal/features/projects/commands/update"
	"pmt/internal/features/projects/domain"
	"pmt/internal/features/projects/queries/checkplanning"
	"pmt/internal/features/projects/queries/checkworking"
	"pmt/internal/features/projects/queries/get"
	"pmt/internal/features/projects/queries/list"
	"pmt/internal/features/projects/queries/liststatuschanges"
	"pmt/internal/features/projects/store"
	"pmt/internal/features/projects/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
func Register(mux *http.ServeMux, tools *sdk.Server, db store.DB) {
	repo := store.NewProjects(db)
	statusChanges := store.NewStatusChanges(db)

	createProject := create.New(repo)
	createProject.Register(mux)
	createProject.RegisterTool(tools)

	changeStatus := changestatus.New(struct {
		*store.Projects
		*store.StatusChanges
	}{repo, statusChanges})
	changeStatus.Register(mux)
	changeStatus.RegisterTool(tools)

	updateProject := update.New(repo)
	updateProject.Register(mux)
	updateProject.RegisterTool(tools)

	deleteProject := delete.New(repo)
	deleteProject.Register(mux)
	deleteProject.RegisterTool(tools)

	get.New(repo).Register(mux)

	listProjects := list.New(repo)
	listProjects.Register(mux)
	listProjects.RegisterTool(tools)

	listStatusChanges := liststatuschanges.New(statusChanges)
	listStatusChanges.Register(mux)
	listStatusChanges.RegisterTool(tools)
}

// Checker permite a otras features saber qué admite un proyecto según su status.
type Checker struct {
	checkPlanning *checkplanning.Handler
	checkWorking  *checkworking.Handler
}

// NewChecker crea el verificador sobre las queries checkplanning y checkworking.
func NewChecker(db sqlcgen.DBTX) *Checker {
	repo := store.NewProjects(db)
	return &Checker{checkPlanning: checkplanning.New(repo), checkWorking: checkworking.New(repo)}
}

// EnsurePlanning devuelve un error de negocio si el proyecto no existe o no está en planning.
func (c *Checker) EnsurePlanning(ctx context.Context, id uuid.UUID) error {
	return c.checkPlanning.Handle(ctx, id)
}

// IsWorking indica si el proyecto está en working. Si no existe, devuelve un error de negocio.
func (c *Checker) IsWorking(ctx context.Context, id uuid.UUID) (bool, error) {
	return c.checkWorking.Handle(ctx, id)
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
