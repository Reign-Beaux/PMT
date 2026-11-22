package removemember

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("DELETE /api/projects/{id}/members/{username}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	id, err := rest.PathUUID(r, "id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	err = h.Handle(r.Context(), Command{ProjectID: id, Username: r.PathValue("username"), Actor: rest.ActorFrom(r.Context())})
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
