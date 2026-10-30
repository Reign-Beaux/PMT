// Package stages construye las slices de la feature y registra sus entradas.
package stages

import (
	"net/http"

	"pmt/internal/features/stages/commands/create"
	"pmt/internal/features/stages/commands/delete"
	"pmt/internal/features/stages/commands/update"
	"pmt/internal/features/stages/queries/get"
	"pmt/internal/features/stages/queries/list"
	"pmt/internal/features/stages/store"
	"pmt/internal/features/stages/store/sqlcgen"
)

// Register registra las rutas REST de la feature.
func Register(mux *http.ServeMux, db sqlcgen.DBTX) {
	repo := store.NewStages(db)

	create.New(repo).Register(mux)
	update.New(repo).Register(mux)
	delete.New(repo).Register(mux)
	get.New(repo).Register(mux)
	list.New(repo).Register(mux)
}
