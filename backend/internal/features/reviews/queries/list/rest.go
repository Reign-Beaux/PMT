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
	values := r.URL.Query()
	query := Query{ItemKind: domain.ItemKind(values.Get("item_kind"))}
	var err error
	if raw := values.Get("item_id"); raw != "" {
		if query.ItemID, err = uuid.Parse(raw); err != nil {
			rest.WriteError(w, kernel.Invalid("item_id inválido"))
			return
		}
	}
	if raw := values.Get("stage_id"); raw != "" {
		if query.StageID, err = uuid.Parse(raw); err != nil {
			rest.WriteError(w, kernel.Invalid("stage_id inválido"))
			return
		}
	}
	if raw := values.Get("pending"); raw != "" {
		if query.Pending, err = strconv.ParseBool(raw); err != nil {
			rest.WriteError(w, kernel.Invalid("pending inválido"))
			return
		}
	}

	items, err := h.Handle(r.Context(), query)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
