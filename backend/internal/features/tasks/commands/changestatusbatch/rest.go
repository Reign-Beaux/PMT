package changestatusbatch

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/stages/{stage_id}/tasks/status", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	stageID, err := rest.PathUUID(r, "stage_id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	var cmd Command
	if err := rest.DecodeJSON(w, r, &cmd); err != nil {
		rest.WriteError(w, err)
		return
	}
	cmd.StageID = stageID
	cmd.Actor = rest.ActorFrom(r.Context())

	items, err := h.Handle(r.Context(), cmd)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
