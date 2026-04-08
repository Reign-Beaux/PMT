package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	phaseapp "project-management-tools/internal/application/phase"
	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// PhaseService is the driving port — defined here, in the consumer.
type PhaseService interface {
	Create(ctx context.Context, input phaseapp.CreateInput) (phase.Phase, error)
	GetByID(ctx context.Context, id shared.ID) (phase.Phase, error)
	ListByProject(ctx context.Context, projectID shared.ID) ([]phase.Phase, error)
	Update(ctx context.Context, id shared.ID, input phaseapp.UpdateInput) (phase.Phase, error)
	Delete(ctx context.Context, id shared.ID) error
}

type PhaseHandler struct {
	svc PhaseService
}

func NewPhaseHandler(svc PhaseService) *PhaseHandler {
	return &PhaseHandler{svc: svc}
}

type phaseResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Order       int       `json:"order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toPhaseResponse(p phase.Phase) phaseResponse {
	return phaseResponse{
		ID:          p.ID().String(),
		ProjectID:   p.ProjectID().String(),
		Name:        p.Name().String(),
		Description: p.Description(),
		Order:       p.Order().Value(),
		Status:      string(p.Status()),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

// POST /projects/{projectId}/phases
func (h *PhaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.svc.Create(r.Context(), phaseapp.CreateInput{
		ProjectID:   chi.URLParam(r, "projectId"),
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		writePhaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toPhaseResponse(p))
}

// GET /projects/{projectId}/phases
func (h *PhaseHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := shared.ParseID(chi.URLParam(r, "projectId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	phases, err := h.svc.ListByProject(r.Context(), projectID)
	if err != nil {
		writePhaseError(w, err)
		return
	}

	resp := make([]phaseResponse, 0, len(phases))
	for _, p := range phases {
		resp = append(resp, toPhaseResponse(p))
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /projects/{projectId}/phases/{id}
func (h *PhaseHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid phase id")
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writePhaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toPhaseResponse(p))
}

// PATCH /projects/{projectId}/phases/{id}
func (h *PhaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid phase id")
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.svc.Update(r.Context(), id, phaseapp.UpdateInput{
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		writePhaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toPhaseResponse(p))
}

// DELETE /projects/{projectId}/phases/{id}
func (h *PhaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid phase id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writePhaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writePhaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, phase.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, project.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, phase.ErrInvalidName):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, shared.ErrInvalidID):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
