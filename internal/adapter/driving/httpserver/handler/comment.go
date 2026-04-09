package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	commentapp "project-management-tools/internal/application/comment"
	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

// CommentService is the driving port — defined here, in the consumer.
type CommentService interface {
	Create(ctx context.Context, input commentapp.CreateInput) (comment.Comment, error)
	ListByIssue(ctx context.Context, issueID shared.ID) ([]comment.Comment, error)
	Delete(ctx context.Context, id shared.ID) error
}

type CommentHandler struct {
	svc CommentService
}

func NewCommentHandler(svc CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

type commentResponse struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toCommentResponse(c comment.Comment) commentResponse {
	return commentResponse{
		ID:        c.ID().String(),
		IssueID:   c.IssueID().String(),
		Body:      c.Body().String(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

// POST /projects/{projectId}/issues/{issueId}/comments
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.svc.Create(r.Context(), commentapp.CreateInput{
		IssueID: chi.URLParam(r, "issueId"),
		Body:    body.Body,
	})
	if err != nil {
		writeCommentError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toCommentResponse(c))
}

// GET /projects/{projectId}/issues/{issueId}/comments
func (h *CommentHandler) ListByIssue(w http.ResponseWriter, r *http.Request) {
	issueID, err := shared.ParseID(chi.URLParam(r, "issueId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	comments, err := h.svc.ListByIssue(r.Context(), issueID)
	if err != nil {
		writeCommentError(w, err)
		return
	}

	resp := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		resp = append(resp, toCommentResponse(c))
	}

	writeJSON(w, http.StatusOK, resp)
}

// DELETE /projects/{projectId}/issues/{issueId}/comments/{commentId}
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := shared.ParseID(chi.URLParam(r, "commentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeCommentError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCommentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, comment.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, issue.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, comment.ErrInvalidBody):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, shared.ErrInvalidID):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
