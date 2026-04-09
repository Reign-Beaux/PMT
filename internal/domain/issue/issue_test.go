package issue_test

import (
	"errors"
	"testing"

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
				if iss.Status() != issue.StatusOpen {
					t.Errorf("expected status %q, got %q", issue.StatusOpen, iss.Status())
				}
				if iss.Priority() != issue.PriorityMedium {
					t.Errorf("expected priority %q, got %q", issue.PriorityMedium, iss.Priority())
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

func TestIssue_Transition(t *testing.T) {
	tests := []struct {
		name    string
		from    issue.Status
		to      issue.Status
		wantErr error
	}{
		{name: "open → in_progress allowed", from: issue.StatusOpen, to: issue.StatusInProgress, wantErr: nil},
		{name: "open → closed allowed", from: issue.StatusOpen, to: issue.StatusClosed, wantErr: nil},
		{name: "in_progress → done allowed", from: issue.StatusInProgress, to: issue.StatusDone, wantErr: nil},
		{name: "in_progress → open allowed", from: issue.StatusInProgress, to: issue.StatusOpen, wantErr: nil},
		{name: "done → closed allowed", from: issue.StatusDone, to: issue.StatusClosed, wantErr: nil},
		{name: "open → done not allowed", from: issue.StatusOpen, to: issue.StatusDone, wantErr: issue.ErrInvalidTransition},
		{name: "closed → any not allowed", from: issue.StatusClosed, to: issue.StatusOpen, wantErr: issue.ErrInvalidTransition},
		{name: "done → open not allowed", from: issue.StatusDone, to: issue.StatusOpen, wantErr: issue.ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID := shared.NewID()
			phaseID := shared.NewID()
			title, _ := issue.NewTitle("Some issue")
			iss, _ := issue.New(projectID, &phaseID, title)

			// Force initial status via reconstitution
			iss = issue.Reconstitute(
				iss.ID(), projectID, &phaseID, title, "", tt.from, issue.PriorityMedium,
				iss.CreatedAt(), iss.UpdatedAt(),
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
