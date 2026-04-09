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
	labelHandler *handler.LabelHandler,
	commentHandler *handler.CommentHandler,
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

		// Labels for a project
		r.Route("/{projectId}/labels", func(r chi.Router) {
			r.Post("/", labelHandler.Create)
			r.Get("/", labelHandler.ListByProject)
			r.Patch("/{labelId}", labelHandler.Update)
			r.Delete("/{labelId}", labelHandler.Delete)
		})

		// Backlog issues (not assigned to any phase)
		r.Route("/{projectId}/issues", func(r chi.Router) {
			r.Post("/", issueHandler.BacklogCreate)
			r.Get("/", issueHandler.BacklogList)

			r.Route("/{issueId}", func(r chi.Router) {
				r.Get("/", issueHandler.GetByID)
				r.Patch("/", issueHandler.Update)
				r.Patch("/transition", issueHandler.Transition)
				r.Delete("/", issueHandler.Delete)

				r.Post("/labels", labelHandler.AssignToIssue)
				r.Delete("/labels/{labelId}", labelHandler.RemoveFromIssue)

				r.Post("/comments", commentHandler.Create)
				r.Get("/comments", commentHandler.ListByIssue)
				r.Delete("/comments/{commentId}", commentHandler.Delete)
			})
		})

		// Phase issues
		r.Route("/{projectId}/phases", func(r chi.Router) {
			r.Post("/", phaseHandler.Create)
			r.Get("/", phaseHandler.ListByProject)
			r.Get("/{id}", phaseHandler.GetByID)
			r.Patch("/{id}", phaseHandler.Update)
			r.Delete("/{id}", phaseHandler.Delete)

			r.Route("/{phaseId}/issues", func(r chi.Router) {
				r.Post("/", issueHandler.Create)
				r.Get("/", issueHandler.ListByPhase)

				r.Route("/{issueId}", func(r chi.Router) {
					r.Get("/", issueHandler.GetByID)
					r.Patch("/", issueHandler.Update)
					r.Patch("/transition", issueHandler.Transition)
					r.Delete("/", issueHandler.Delete)

					r.Post("/labels", labelHandler.AssignToIssue)
					r.Delete("/labels/{labelId}", labelHandler.RemoveFromIssue)

					r.Post("/comments", commentHandler.Create)
					r.Get("/comments", commentHandler.ListByIssue)
					r.Delete("/comments/{commentId}", commentHandler.Delete)
				})
			})
		})
	})

	return r
}
