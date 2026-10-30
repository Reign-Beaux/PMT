// Package rest expone el módulo work por REST.
package rest

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/work/domain"
	"pmt/internal/features/work/usecase"
	platformrest "pmt/internal/platform/rest"
)

// Projects atiende las peticiones de proyectos.
type Projects struct {
	projects *usecase.Projects
}

// NewProjects crea el controlador de proyectos.
func NewProjects(projects *usecase.Projects) *Projects {
	return &Projects{projects: projects}
}

// Register registra las rutas de proyectos.
func (h *Projects) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/projects", h.create)
	mux.HandleFunc("GET /api/projects", h.list)
	mux.HandleFunc("GET /api/projects/{id}", h.get)
	mux.HandleFunc("PUT /api/projects/{id}", h.update)
	mux.HandleFunc("DELETE /api/projects/{id}", h.delete)
}

type projectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type projectResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h *Projects) create(w http.ResponseWriter, r *http.Request) {
	var req projectRequest
	if err := platformrest.DecodeJSON(w, r, &req); err != nil {
		platformrest.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.projects.Create(r.Context(), req.Name, req.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	platformrest.WriteJSON(w, http.StatusCreated, toResponse(p))
}

func (h *Projects) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	response := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		response = append(response, toResponse(p))
	}
	platformrest.WriteJSON(w, http.StatusOK, response)
}

func (h *Projects) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	p, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	platformrest.WriteJSON(w, http.StatusOK, toResponse(p))
}

func (h *Projects) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req projectRequest
	if err := platformrest.DecodeJSON(w, r, &req); err != nil {
		platformrest.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.projects.Update(r.Context(), id, req.Name, req.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	platformrest.WriteJSON(w, http.StatusOK, toResponse(p))
}

func (h *Projects) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.projects.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		platformrest.WriteError(w, http.StatusBadRequest, "id inválido")
		return uuid.Nil, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		platformrest.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrProjectNameRequired):
		platformrest.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("error interno: %v", err)
		platformrest.WriteError(w, http.StatusInternalServerError, "error interno")
	}
}

func toResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
