package getinvitation

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/invitations/{token}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	status, err := h.Handle(r.Context(), r.PathValue("token"))
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, status)
}
