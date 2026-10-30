package delete

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("DELETE /api/tasks/{id}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	id, err := rest.PathUUID(r, "id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	if err := h.Handle(r.Context(), id); err != nil {
		rest.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
