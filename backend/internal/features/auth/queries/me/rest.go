package me

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/me", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	user, err := h.Handle(r.Context(), rest.ActorFrom(r.Context()))
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, user)
}
