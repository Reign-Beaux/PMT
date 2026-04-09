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
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// IssueService is the driving port — defined here, in the consumer.
type IssueService interface {
	Create(ctx context.Context, input issueapp.CreateInput) (issue.Issue, error)
	GetByID(ctx context.Context, id shared.ID) (issue.Issue, error)
	ListByPhase(ctx context.Context, phaseID shared.ID) ([]issue.Issue, error)
	ListBacklog(ctx context.Context, projectID shared.ID) ([]issue.Issue, error)
	Update(ctx context.Context, id shared.ID, input issueapp.UpdateInput) (issue.Issue, error)
	Transition(ctx context.Context, id shared.ID, status string) (issue.Issue, error)
	Delete(ctx context.Context, id shared.ID) error
	AddLabel(ctx context.Context, issueID, labelID shared.ID) error
	RemoveLabel(ctx context.Context, issueID, labelID shared.ID) error
}

type IssueHandler struct {
	svc IssueService
}

func NewIssueHandler(svc IssueService) *IssueHandler {
	return &IssueHandler{svc: svc}
}

type issueResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	PhaseID   *string   `json:"phase_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Spec      string    `json:"spec"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	DueDate   *string   `json:"due_date"`
	LabelIDs  []string  `json:"label_ids"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toIssueResponse(i issue.Issue) issueResponse {
	var phaseID *string
	if i.PhaseID() != nil {
		s := i.PhaseID().String()
		phaseID = &s
	}
	var dueDate *string
	if i.DueDate() != nil {
		s := i.DueDate().Format(time.RFC3339)
		dueDate = &s
	}
	labelIDs := make([]string, 0, len(i.LabelIDs()))
	for _, id := range i.LabelIDs() {
		labelIDs = append(labelIDs, id.String())
	}
	return issueResponse{
		ID:        i.ID().String(),
		ProjectID: i.ProjectID().String(),
		PhaseID:   phaseID,
		Type:      string(i.Type()),
		Title:     i.Title().String(),
		Spec:      i.Spec(),
		Status:    string(i.Status()),
		Priority:  string(i.Priority()),
		DueDate:   dueDate,
		LabelIDs:  labelIDs,
		CreatedAt: i.CreatedAt(),
		UpdatedAt: i.UpdatedAt(),
	}
}

// issueCreateBody is the shared request body for issue creation.
type issueCreateBody struct {
	Title    string  `json:"title"`
	Spec     string  `json:"spec"`
	Priority string  `json:"priority"`
	Type     string  `json:"type"`
	DueDate  *string `json:"due_date"`
}

func parseDueDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// POST /projects/{projectId}/phases/{phaseId}/issues
func (h *IssueHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body issueCreateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dueDate, err := parseDueDate(body.DueDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "due_date must be RFC3339 format")
		return
	}

	phaseIDStr := chi.URLParam(r, "phaseId")
	iss, err := h.svc.Create(r.Context(), issueapp.CreateInput{
		ProjectID: chi.URLParam(r, "projectId"),
		PhaseID:   &phaseIDStr,
		Title:     body.Title,
		Spec:      body.Spec,
		Priority:  body.Priority,
		Type:      body.Type,
		DueDate:   dueDate,
	})
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toIssueResponse(iss))
}

// POST /projects/{projectId}/issues
func (h *IssueHandler) BacklogCreate(w http.ResponseWriter, r *http.Request) {
	var body issueCreateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dueDate, err := parseDueDate(body.DueDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "due_date must be RFC3339 format")
		return
	}

	iss, err := h.svc.Create(r.Context(), issueapp.CreateInput{
		ProjectID: chi.URLParam(r, "projectId"),
		PhaseID:   nil,
		Title:     body.Title,
		Spec:      body.Spec,
		Priority:  body.Priority,
		Type:      body.Type,
		DueDate:   dueDate,
	})
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toIssueResponse(iss))
}

// GET /projects/{projectId}/issues
func (h *IssueHandler) BacklogList(w http.ResponseWriter, r *http.Request) {
	projectID, err := shared.ParseID(chi.URLParam(r, "projectId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	issues, err := h.svc.ListBacklog(r.Context(), projectID)
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

// GET .../issues/{issueId}
func (h *IssueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "issueId"))
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

// PATCH .../issues/{issueId}
func (h *IssueHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "issueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	var body struct {
		Title        *string         `json:"title"`
		Spec         *string         `json:"spec"`
		Priority     *string         `json:"priority"`
		Type         *string         `json:"type"`
		DueDate      json.RawMessage `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := issueapp.UpdateInput{
		Title:    body.Title,
		Spec:     body.Spec,
		Priority: body.Priority,
		Type:     body.Type,
	}

	// due_date: absent = no change; null = clear; "..." = set
	if body.DueDate != nil {
		if string(body.DueDate) == "null" {
			input.ClearDueDate = true
		} else {
			var dueDateStr string
			if err := json.Unmarshal(body.DueDate, &dueDateStr); err != nil {
				writeError(w, http.StatusBadRequest, "due_date must be RFC3339 format or null")
				return
			}
			t, err := time.Parse(time.RFC3339, dueDateStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "due_date must be RFC3339 format or null")
				return
			}
			input.DueDate = &t
		}
	}

	iss, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		writeIssueError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toIssueResponse(iss))
}

// PATCH .../issues/{issueId}/transition
func (h *IssueHandler) Transition(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "issueId"))
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

// DELETE .../issues/{issueId}
func (h *IssueHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "issueId"))
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
	case errors.Is(err, project.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, issue.ErrInvalidTitle):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, issue.ErrInvalidTransition):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, issue.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, issue.ErrInvalidPriority):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, issue.ErrInvalidType):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, shared.ErrInvalidID):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
