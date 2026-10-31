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
	"pmt/internal/features/projects/queries/checkplanning"
	"pmt/internal/features/projects/queries/get"
	"pmt/internal/features/projects/queries/list"
	"pmt/internal/features/projects/store"
	"pmt/internal/features/projects/store/sqlcgen"
)

// Register registra las rutas REST y las herramientas MCP de la feature.
func Register(mux *http.ServeMux, tools *sdk.Server, db sqlcgen.DBTX) {
	repo := store.NewProjects(db)

	createProject := create.New(repo)
	createProject.Register(mux)
	createProject.RegisterTool(tools)

	changeStatus := changestatus.New(repo)
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
}

// Checker permite a otras features saber si un proyecto admite cambios de definición.
type Checker struct {
	checkPlanning *checkplanning.Handler
}

// NewChecker crea el verificador sobre la query checkplanning.
func NewChecker(db sqlcgen.DBTX) *Checker {
	return &Checker{checkPlanning: checkplanning.New(store.NewProjects(db))}
}

// EnsurePlanning devuelve un error de negocio si el proyecto no existe o no está en planning.
func (c *Checker) EnsurePlanning(ctx context.Context, id uuid.UUID) error {
	return c.checkPlanning.Handle(ctx, id)
}
