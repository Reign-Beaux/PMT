package list

import (
	"net/http"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/stages/{stage_id}/tasks", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	stageID, err := rest.PathUUID(r, "stage_id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	items, err := h.Handle(r.Context(), stageID, domain.Status(r.URL.Query().Get("status")))
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
