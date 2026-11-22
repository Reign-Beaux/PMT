package list

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/profiles", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, _ *http.Request) {
	rest.WriteJSON(w, http.StatusOK, h.Handle())
}
