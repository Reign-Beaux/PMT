// Package projects construye las slices de la feature y registra sus rutas.
package projects

import (
	"net/http"

	"pmt/internal/features/projects/commands/create"
	"pmt/internal/features/projects/commands/delete"
	"pmt/internal/features/projects/commands/update"
	"pmt/internal/features/projects/queries/get"
	"pmt/internal/features/projects/queries/list"
	"pmt/internal/features/projects/store"
	"pmt/internal/features/projects/store/sqlcgen"
)

// Register registra las rutas REST de la feature.
func Register(mux *http.ServeMux, db sqlcgen.DBTX) {
	repo := store.NewProjects(db)

	create.New(repo).Register(mux)
	update.New(repo).Register(mux)
	delete.New(repo).Register(mux)
	get.New(repo).Register(mux)
	list.New(repo).Register(mux)
}
