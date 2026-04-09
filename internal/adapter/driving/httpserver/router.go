package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"project-management-tools/internal/adapter/driving/httpserver/handler"
)

func NewRouter(
	projectHandler *handler.ProjectHandler,
	phaseHandler *handler.PhaseHandler,
	issueHandler *handler.IssueHandler,
) http.Handler {
	r := chi.NewRouter()

	healthHandler := handler.NewHealthHandler()
	r.Get("/health", healthHandler.Handle)

	r.Route("/projects", func(r chi.Router) {
		r.Post("/", projectHandler.Create)
		r.Get("/", projectHandler.List)
		r.Get("/{id}", projectHandler.GetByID)
		r.Patch("/{id}", projectHandler.Update)
		r.Delete("/{id}", projectHandler.Delete)

		r.Route("/{projectId}/issues", func(r chi.Router) {
			r.Post("/", issueHandler.BacklogCreate)
			r.Get("/", issueHandler.BacklogList)
		})

		r.Route("/{projectId}/phases", func(r chi.Router) {
			r.Post("/", phaseHandler.Create)
			r.Get("/", phaseHandler.ListByProject)
			r.Get("/{id}", phaseHandler.GetByID)
			r.Patch("/{id}", phaseHandler.Update)
			r.Delete("/{id}", phaseHandler.Delete)

			r.Route("/{phaseId}/issues", func(r chi.Router) {
				r.Post("/", issueHandler.Create)
				r.Get("/", issueHandler.ListByPhase)
				r.Get("/{id}", issueHandler.GetByID)
				r.Patch("/{id}", issueHandler.Update)
				r.Patch("/{id}/transition", issueHandler.Transition)
				r.Delete("/{id}", issueHandler.Delete)
			})
		})
	})

	return r
}
