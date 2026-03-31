package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"project-management-tools/internal/adapter/driving/httpserver/handler"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	healthHandler := handler.NewHealthHandler()

	r.Get("/health", healthHandler.Handle)

	return r
}
