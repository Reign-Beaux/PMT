package getclilogin

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/cli/login/{code}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	item, err := h.Handle(r.Context(), Query{Code: r.PathValue("code"), Actor: rest.ActorFrom(r.Context())})
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, item)
}
