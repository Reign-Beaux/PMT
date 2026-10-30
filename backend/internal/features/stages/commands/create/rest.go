package create

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/projects/{project_id}/stages", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	projectID, err := rest.PathUUID(r, "project_id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	var cmd Command
	if err := rest.DecodeJSON(w, r, &cmd); err != nil {
		rest.WriteError(w, err)
		return
	}
	cmd.ProjectID = projectID

	result, err := h.Handle(r.Context(), cmd)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusCreated, result)
}
