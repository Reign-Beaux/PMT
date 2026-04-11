package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"project-management-tools/internal/adapter/driving/httpserver/middleware"
	projectapp "project-management-tools/internal/application/project"
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// ProjectService is the driving port — defined here, in the consumer.
type ProjectService interface {
	Create(ctx context.Context, input projectapp.CreateInput) (project.Project, error)
	GetByID(ctx context.Context, id shared.ID) (project.Project, error)
	List(ctx context.Context, ownerID shared.ID) ([]project.Project, error)
	Update(ctx context.Context, id shared.ID, input projectapp.UpdateInput) (project.Project, error)
	Delete(ctx context.Context, id shared.ID) error
}

type ProjectHandler struct {
	svc ProjectService
}

func NewProjectHandler(svc ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// projectResponse is the JSON shape returned to the client.
type projectResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toProjectResponse(p project.Project) projectResponse {
	return projectResponse{
		ID:          p.ID().String(),
		Name:        p.Name().String(),
		Description: p.Description(),
		Status:      string(p.Status()),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

// POST /projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.svc.Create(r.Context(), projectapp.CreateInput{
		OwnerID:     userID,
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		writeProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toProjectResponse(p))
}

// GET /projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projects, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}

	resp := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, toProjectResponse(p))
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /projects/{id}
func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

// PATCH /projects/{id}
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := projectapp.UpdateInput{
		Name:        body.Name,
		Description: body.Description,
	}
	if body.Status != nil {
		s, err := project.ParseStatus(*body.Status)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		input.Status = &s
	}

	p, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		writeProjectError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

// DELETE /projects/{id}
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeProjectError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeProjectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, project.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, project.ErrInvalidName),
		errors.Is(err, project.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
