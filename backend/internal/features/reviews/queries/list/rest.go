package list

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/findings", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	itemID, err := uuid.Parse(query.Get("item_id"))
	if err != nil {
		rest.WriteError(w, kernel.Invalid("item_id inválido"))
		return
	}
	pending := false
	if raw := query.Get("pending"); raw != "" {
		if pending, err = strconv.ParseBool(raw); err != nil {
			rest.WriteError(w, kernel.Invalid("pending inválido"))
			return
		}
	}

	items, err := h.Handle(r.Context(), domain.ItemKind(query.Get("item_kind")), itemID, pending)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
