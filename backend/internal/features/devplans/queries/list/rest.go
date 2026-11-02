package list

import (
	"net/http"

	"pmt/internal/platform/rest"
)

// Register registra la ruta de la operación.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dev-plans", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	var query Query
	var err error
	if query.StageID, err = parseOptional(values.Get("stage_id"), "stage_id inválido"); err != nil {
		rest.WriteError(w, err)
		return
	}
	if query.TaskID, err = parseOptional(values.Get("task_id"), "task_id inválido"); err != nil {
		rest.WriteError(w, err)
		return
	}
	if query.BugID, err = parseOptional(values.Get("bug_id"), "bug_id inválido"); err != nil {
		rest.WriteError(w, err)
		return
	}

	items, err := h.Handle(r.Context(), query)
	if err != nil {
		rest.WriteError(w, err)
		return
	}
	rest.WriteJSON(w, http.StatusOK, items)
}
