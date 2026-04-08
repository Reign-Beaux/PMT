package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"project-management-tools/internal/adapter/driving/httpserver/handler"
)

func NewRouter(projectHandler *handler.ProjectHandler) http.Handler {
	r := chi.NewRouter()

	healthHandler := handler.NewHealthHandler()
	r.Get("/health", healthHandler.Handle)

	r.Route("/projects", func(r chi.Router) {
		r.Post("/", projectHandler.Create)
		r.Get("/", projectHandler.List)
		r.Get("/{id}", projectHandler.GetByID)
		r.Patch("/{id}", projectHandler.Update)
		r.Delete("/{id}", projectHandler.Delete)
	})

	return r
}
