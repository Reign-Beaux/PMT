package list

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects/{project_id}/stages", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	projectID, err := rest.PathUUID(r, "project_id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	items, err := h.Handle(r.Context(), projectID)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
