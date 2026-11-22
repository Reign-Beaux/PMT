package get

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/profiles/{profile}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	text, err := h.Handle(r.PathValue("profile"))
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(text))
}
