package comment_test

import (
	"errors"
	"testing"

	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/shared"
)

func TestNewBody(t *testing.T) {
	tests := []struct {
		input   string
		wantErr error
	}{
		{input: "This is a comment", wantErr: nil},
		{input: "  trimmed  ", wantErr: nil},
		{input: "", wantErr: comment.ErrInvalidBody},
		{input: "   ", wantErr: comment.ErrInvalidBody},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := comment.NewBody(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	issueID := shared.NewID()
	body, _ := comment.NewBody("Investigating the root cause")

	c, err := comment.New(issueID, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID().IsZero() {
		t.Error("expected non-zero ID")
	}
	if c.IssueID() != issueID {
		t.Error("issue ID mismatch")
	}
	if c.Body().String() != "Investigating the root cause" {
		t.Errorf("unexpected body: %q", c.Body().String())
	}
}
