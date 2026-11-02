package listchanges

import (
	"net/http"

	"github.com/google/uuid"

	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects/{project_id}/stage-changes", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	projectID, err := rest.PathUUID(r, "project_id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}

	var stageID *uuid.UUID
	if raw := r.URL.Query().Get("stage_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			rest.WriteError(w, kernel.Invalid("stage_id inválido"))
			return
		}
		stageID = &id
	}

	items, err := h.Handle(r.Context(), projectID, stageID)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
