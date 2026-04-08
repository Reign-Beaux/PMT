package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	issueapp "project-management-tools/internal/application/issue"
	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/shared"
)

// IssueService is the driving port — defined here, in the consumer.
type IssueService interface {
	Create(ctx context.Context, input issueapp.CreateInput) (issue.Issue, error)
	GetByID(ctx context.Context, id shared.ID) (issue.Issue, error)
	ListByPhase(ctx context.Context, phaseID shared.ID) ([]issue.Issue, error)
	Update(ctx context.Context, id shared.ID, input issueapp.UpdateInput) (issue.Issue, error)
	Transition(ctx context.Context, id shared.ID, status string) (issue.Issue, error)
	Delete(ctx context.Context, id shared.ID) error
}

type IssueHandler struct {
	svc IssueService
}

func NewIssueHandler(svc IssueService) *IssueHandler {
	return &IssueHandler{svc: svc}
}

type issueResponse struct {
	ID        string    `json:"id"`
	PhaseID   string    `json:"phase_id"`
	Title     string    `json:"title"`
	Spec      string    `json:"spec"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toIssueResponse(i issue.Issue) issueResponse {
	return issueResponse{
		ID:        i.ID().String(),
		PhaseID:   i.PhaseID().String(),
		Title:     i.Title().String(),
		Spec:      i.Spec(),
		Status:    string(i.Status()),
		Priority:  string(i.Priority()),
		CreatedAt: i.CreatedAt(),
		UpdatedAt: i.UpdatedAt(),
	}
}

// POST /projects/{projectId}/phases/{phaseId}/issues
func (h *IssueHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title    string `json:"title"`
		Spec     string `json:"spec"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	iss, err := h.svc.Create(r.Context(), issueapp.CreateInput{
		PhaseID:  chi.URLParam(r, "phaseId"),
		Title:    body.Title,
		Spec:     body.Spec,
		Priority: body.Priority,
	})
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toIssueResponse(iss))
}

// GET /projects/{projectId}/phases/{phaseId}/issues
func (h *IssueHandler) ListByPhase(w http.ResponseWriter, r *http.Request) {
	phaseID, err := shared.ParseID(chi.URLParam(r, "phaseId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid phase id")
		return
	}

	issues, err := h.svc.ListByPhase(r.Context(), phaseID)
	if err != nil {
		writeIssueError(w, err)
		return
	}

	resp := make([]issueResponse, 0, len(issues))
	for _, iss := range issues {
		resp = append(resp, toIssueResponse(iss))
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /projects/{projectId}/phases/{phaseId}/issues/{id}
func (h *IssueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	iss, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toIssueResponse(iss))
}

// PATCH /projects/{projectId}/phases/{phaseId}/issues/{id}
func (h *IssueHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	var body struct {
		Title    *string `json:"title"`
		Spec     *string `json:"spec"`
		Priority *string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	iss, err := h.svc.Update(r.Context(), id, issueapp.UpdateInput{
		Title:    body.Title,
		Spec:     body.Spec,
		Priority: body.Priority,
	})
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toIssueResponse(iss))
}

// PATCH /projects/{projectId}/phases/{phaseId}/issues/{id}/transition
func (h *IssueHandler) Transition(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	iss, err := h.svc.Transition(r.Context(), id, body.Status)
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toIssueResponse(iss))
}

// DELETE /projects/{projectId}/phases/{phaseId}/issues/{id}
func (h *IssueHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeIssueError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeIssueError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, issue.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, phase.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, issue.ErrInvalidTitle):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, issue.ErrInvalidTransition):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, issue.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, issue.ErrInvalidPriority):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, shared.ErrInvalidID):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
