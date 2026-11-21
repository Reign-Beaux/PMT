package acceptinvitation

import (
	"net/http"

	"pmt/internal/features/auth/domain"
	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación. Es pública: la invitación es la credencial.
func (h *Handler) Register(mux *http.ServeMux, secure bool) {
	mux.HandleFunc("POST /api/invitations/{token}/accept", func(w http.ResponseWriter, r *http.Request) {
		var cmd Command
		if err := rest.DecodeJSON(w, r, &cmd); err != nil {
			rest.WriteError(w, err)
			return
		}
		cmd.Token = r.PathValue("token")
		result, err := h.Handle(r.Context(), cmd)
		if err != nil {
			rest.WriteError(w, err)
			return
		}
		rest.SetSessionCookie(w, result.Session, int(domain.SessionTTL.Seconds()), secure)
		rest.WriteJSON(w, http.StatusCreated, result.User)
	})
}
