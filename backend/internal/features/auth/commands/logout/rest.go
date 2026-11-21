package logout

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación. Es pública: con una sesión vencida
// también borra la cookie.
func (h *Handler) Register(mux *http.ServeMux, secure bool) {
	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if err := h.Handle(r.Context(), rest.SessionFrom(r)); err != nil {
			rest.WriteError(w, err)
			return
		}
		rest.SetSessionCookie(w, "", -1, secure)
		w.WriteHeader(http.StatusNoContent)
	})
}
