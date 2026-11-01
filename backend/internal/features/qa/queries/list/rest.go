package list

import (
	"net/http"

	"github.com/google/uuid"

	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/stages/{stage_id}/test-plans", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	stageID, err := rest.PathUUID(r, "stage_id")
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	var taskID *uuid.UUID
	if raw := r.URL.Query().Get("task_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			rest.WriteError(w, kernel.Invalid("task_id inválido"))
			return
		}
		taskID = &parsed
	}

	items, err := h.Handle(r.Context(), stageID, taskID)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
