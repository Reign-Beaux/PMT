package changepassword

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/me/password", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	var cmd Command
	if err := rest.DecodeJSON(w, r, &cmd); err != nil {
		rest.WriteError(w, err)
		return
	}
	cmd.Actor = rest.ActorFrom(r.Context())
	if _, bearer := r.Header["Authorization"]; !bearer {
		cmd.Session = rest.SessionFrom(r)
	}
	if err := h.Handle(r.Context(), cmd); err != nil {
		rest.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
