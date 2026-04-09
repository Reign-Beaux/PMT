package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	labelapp "project-management-tools/internal/application/label"
	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// LabelService is the driving port — defined here, in the consumer.
type LabelService interface {
	Create(ctx context.Context, input labelapp.CreateInput) (label.Label, error)
	GetByID(ctx context.Context, id shared.ID) (label.Label, error)
	ListByProject(ctx context.Context, projectID shared.ID) ([]label.Label, error)
	Update(ctx context.Context, id shared.ID, input labelapp.UpdateInput) (label.Label, error)
	Delete(ctx context.Context, id shared.ID) error
}

// IssueLabeler is the subset of IssueService used for label assignment.
// Defined here, in the consumer.
type IssueLabeler interface {
	AddLabel(ctx context.Context, issueID, labelID shared.ID) error
	RemoveLabel(ctx context.Context, issueID, labelID shared.ID) error
}

type LabelHandler struct {
	svc      LabelService
	issueSvc IssueLabeler
}

func NewLabelHandler(svc LabelService, issueSvc IssueLabeler) *LabelHandler {
	return &LabelHandler{svc: svc, issueSvc: issueSvc}
}

type labelResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toLabelResponse(l label.Label) labelResponse {
	return labelResponse{
		ID:        l.ID().String(),
		ProjectID: l.ProjectID().String(),
		Name:      l.Name().String(),
		Color:     l.Color().String(),
		CreatedAt: l.CreatedAt(),
		UpdatedAt: l.UpdatedAt(),
	}
}

// POST /projects/{projectId}/labels
func (h *LabelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	l, err := h.svc.Create(r.Context(), labelapp.CreateInput{
		ProjectID: chi.URLParam(r, "projectId"),
		Name:      body.Name,
		Color:     body.Color,
	})
	if err != nil {
		writeLabelError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toLabelResponse(l))
}

// GET /projects/{projectId}/labels
func (h *LabelHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := shared.ParseID(chi.URLParam(r, "projectId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	labels, err := h.svc.ListByProject(r.Context(), projectID)
	if err != nil {
		writeLabelError(w, err)
		return
	}

	resp := make([]labelResponse, 0, len(labels))
	for _, l := range labels {
		resp = append(resp, toLabelResponse(l))
	}

	writeJSON(w, http.StatusOK, resp)
}

// PATCH /projects/{projectId}/labels/{labelId}
func (h *LabelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "labelId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid label id")
		return
	}

	var body struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	l, err := h.svc.Update(r.Context(), id, labelapp.UpdateInput{
		Name:  body.Name,
		Color: body.Color,
	})
	if err != nil {
		writeLabelError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toLabelResponse(l))
}

// DELETE /projects/{projectId}/labels/{labelId}
func (h *LabelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "labelId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid label id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeLabelError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /projects/{projectId}/issues/{issueId}/labels
func (h *LabelHandler) AssignToIssue(w http.ResponseWriter, r *http.Request) {
	issueID, err := shared.ParseID(chi.URLParam(r, "issueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	var body struct {
		LabelID string `json:"label_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	labelID, err := shared.ParseID(body.LabelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid label id")
		return
	}

	if err := h.issueSvc.AddLabel(r.Context(), issueID, labelID); err != nil {
		writeLabelError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /projects/{projectId}/issues/{issueId}/labels/{labelId}
func (h *LabelHandler) RemoveFromIssue(w http.ResponseWriter, r *http.Request) {
	issueID, err := shared.ParseID(chi.URLParam(r, "issueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	labelID, err := shared.ParseID(chi.URLParam(r, "labelId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid label id")
		return
	}

	if err := h.issueSvc.RemoveLabel(r.Context(), issueID, labelID); err != nil {
		writeLabelError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeLabelError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, label.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, project.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, label.ErrInvalidName):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, label.ErrInvalidColor):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, label.ErrDuplicateName):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, shared.ErrInvalidID):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
