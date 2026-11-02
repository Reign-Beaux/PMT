package list

import (
	"net/http"

	"github.com/google/uuid"

	"pmt/internal/features/activity/domain"
	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/comments", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	itemID, err := uuid.Parse(values.Get("item_id"))
	if err != nil {
		rest.WriteError(w, kernel.Invalid("item_id inválido"))
		return
	}

	items, err := h.Handle(r.Context(), Query{ItemKind: domain.ItemKind(values.Get("item_kind")), ItemID: itemID})
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
