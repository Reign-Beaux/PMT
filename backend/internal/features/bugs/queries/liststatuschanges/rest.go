package liststatuschanges

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/bugs/{id}/status-changes", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	id, err := rest.PathUUID(r, "id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	items, err := h.Handle(r.Context(), id)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
