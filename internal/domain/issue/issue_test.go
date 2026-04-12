package issue_test

import (
	"errors"
	"testing"
	"time"

	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

func TestNewTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid title", input: "Fix login bug", wantErr: nil},
		{name: "trimmed spaces", input: "  Fix login bug  ", wantErr: nil},
		{name: "empty string", input: "", wantErr: issue.ErrInvalidTitle},
		{name: "only spaces", input: "   ", wantErr: issue.ErrInvalidTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := issue.NewTitle(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseIssueType(t *testing.T) {
	tests := []struct {
		input   string
		wantErr error
	}{
		{input: "task", wantErr: nil},
		{input: "bug", wantErr: nil},
		{input: "feature", wantErr: nil},
		{input: "improvement", wantErr: nil},
		{input: "unknown", wantErr: issue.ErrInvalidType},
		{input: "", wantErr: issue.ErrInvalidType},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := issue.ParseIssueType(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	validProjectID := shared.NewID()
	validPhaseID := shared.NewID()
	validTitle, _ := issue.NewTitle("Fix login bug")

	tests := []struct {
		name      string
		projectID shared.ID
		phaseID   *shared.ID
		title     issue.Title
		wantErr   error
	}{
		{
			name:      "phase-scoped issue",
			projectID: validProjectID,
			phaseID:   &validPhaseID,
			title:     validTitle,
			wantErr:   nil,
		},
		{
			name:      "backlog issue (nil phase)",
			projectID: validProjectID,
			phaseID:   nil,
			title:     validTitle,
			wantErr:   nil,
		},
		{
			name:      "zero project id rejected",
			projectID: shared.ID{},
			phaseID:   nil,
			title:     validTitle,
			wantErr:   issue.ErrInvalidProjectID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iss, err := issue.New(tt.projectID, tt.phaseID, tt.title)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if iss.ID().IsZero() {
					t.Error("expected non-zero ID")
				}
				if iss.Type() != issue.IssueTypeTask {
					t.Errorf("expected default type %q, got %q", issue.IssueTypeTask, iss.Type())
				}
				if iss.Status() != issue.StatusOpen {
					t.Errorf("expected status %q, got %q", issue.StatusOpen, iss.Status())
				}
				if iss.Priority() != issue.PriorityMedium {
					t.Errorf("expected priority %q, got %q", issue.PriorityMedium, iss.Priority())
				}
				if iss.DueDate() != nil {
					t.Error("expected nil due date by default")
				}
				if tt.phaseID == nil && !iss.IsBacklog() {
					t.Error("expected issue to be in backlog")
				}
				if tt.phaseID != nil && iss.IsBacklog() {
					t.Error("expected issue to not be in backlog")
				}
			}
		})
	}
}

func TestIssue_SetDueDate(t *testing.T) {
	projectID := shared.NewID()
	title, _ := issue.NewTitle("Some issue")
	iss, _ := issue.New(projectID, nil, title)

	due := time.Now().Add(24 * time.Hour)
	iss.SetDueDate(&due)
	if iss.DueDate() == nil {
		t.Fatal("expected due date to be set")
	}

	iss.SetDueDate(nil)
	if iss.DueDate() != nil {
		t.Fatal("expected due date to be cleared")
	}
}

func TestIssue_Transition(t *testing.T) {
	tests := []struct {
		name    string
		from    issue.Status
		to      issue.Status
		wantErr error
	}{
		// Valid transitions
		{name: "open → in_progress", from: issue.StatusOpen, to: issue.StatusInProgress, wantErr: nil},
		{name: "open → canceled", from: issue.StatusOpen, to: issue.StatusCanceled, wantErr: nil},
		{name: "in_progress → done", from: issue.StatusInProgress, to: issue.StatusDone, wantErr: nil},
		{name: "in_progress → stopped", from: issue.StatusInProgress, to: issue.StatusStopped, wantErr: nil},
		{name: "in_progress → canceled", from: issue.StatusInProgress, to: issue.StatusCanceled, wantErr: nil},
		{name: "stopped → in_progress", from: issue.StatusStopped, to: issue.StatusInProgress, wantErr: nil},
		{name: "stopped → canceled", from: issue.StatusStopped, to: issue.StatusCanceled, wantErr: nil},
		{name: "done → in_progress (QA reject)", from: issue.StatusDone, to: issue.StatusInProgress, wantErr: nil},
		{name: "done → closed (QA approve)", from: issue.StatusDone, to: issue.StatusClosed, wantErr: nil},
		// Invalid transitions
		{name: "open → closed not allowed", from: issue.StatusOpen, to: issue.StatusClosed, wantErr: issue.ErrInvalidTransition},
		{name: "open → done not allowed", from: issue.StatusOpen, to: issue.StatusDone, wantErr: issue.ErrInvalidTransition},
		{name: "open → stopped not allowed", from: issue.StatusOpen, to: issue.StatusStopped, wantErr: issue.ErrInvalidTransition},
		{name: "in_progress → open not allowed", from: issue.StatusInProgress, to: issue.StatusOpen, wantErr: issue.ErrInvalidTransition},
		{name: "in_progress → closed not allowed", from: issue.StatusInProgress, to: issue.StatusClosed, wantErr: issue.ErrInvalidTransition},
		{name: "stopped → done not allowed", from: issue.StatusStopped, to: issue.StatusDone, wantErr: issue.ErrInvalidTransition},
		{name: "stopped → closed not allowed", from: issue.StatusStopped, to: issue.StatusClosed, wantErr: issue.ErrInvalidTransition},
		{name: "done → open not allowed", from: issue.StatusDone, to: issue.StatusOpen, wantErr: issue.ErrInvalidTransition},
		{name: "done → stopped not allowed", from: issue.StatusDone, to: issue.StatusStopped, wantErr: issue.ErrInvalidTransition},
		{name: "done → canceled not allowed", from: issue.StatusDone, to: issue.StatusCanceled, wantErr: issue.ErrInvalidTransition},
		{name: "closed → any not allowed", from: issue.StatusClosed, to: issue.StatusOpen, wantErr: issue.ErrInvalidTransition},
		{name: "closed → in_progress not allowed", from: issue.StatusClosed, to: issue.StatusInProgress, wantErr: issue.ErrInvalidTransition},
		{name: "canceled → any not allowed", from: issue.StatusCanceled, to: issue.StatusOpen, wantErr: issue.ErrInvalidTransition},
		{name: "canceled → in_progress not allowed", from: issue.StatusCanceled, to: issue.StatusInProgress, wantErr: issue.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID := shared.NewID()
			phaseID := shared.NewID()
			title, _ := issue.NewTitle("Some issue")
			iss, _ := issue.New(projectID, &phaseID, title)

			iss = issue.Reconstitute(
				iss.ID(), projectID, &phaseID, issue.IssueTypeTask,
				title, "", tt.from, issue.PriorityMedium,
				nil, nil, iss.CreatedAt(), iss.UpdatedAt(),
			)

			err := iss.Transition(tt.to)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && iss.Status() != tt.to {
				t.Errorf("expected status %q, got %q", tt.to, iss.Status())
			}
		})
	}
}
