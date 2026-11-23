package authorizeclilogin

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/cli/login/{code}/authorize", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	item, err := h.Handle(r.Context(), Command{Code: r.PathValue("code"), Actor: rest.ActorFrom(r.Context())})
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, item)
}
