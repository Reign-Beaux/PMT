package updatemember

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/projects/{id}/members/{username}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	id, err := rest.PathUUID(r, "id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	var cmd Command
	if err := rest.DecodeJSON(w, r, &cmd); err != nil {
		rest.WriteError(w, err)
		return
	}
	cmd.ProjectID = id
	cmd.Username = r.PathValue("username")
	cmd.Actor = rest.ActorFrom(r.Context())

	item, err := h.Handle(r.Context(), cmd)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, item)
}
