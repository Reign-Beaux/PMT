package workflow

import (
	"context"
	"fmt"
	"strings"

	"project-management-tools/internal/application/notification"
	"project-management-tools/internal/application/uow"
	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/shared"
)

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(repos uow.Repositories) error) error
}

type Service struct {
	uow      UnitOfWork
	notifier notification.Notifier
}

func NewService(uow UnitOfWork, notifier notification.Notifier) *Service {
	return &Service{uow: uow, notifier: notifier}
}

type FindingInput struct {
	Title string
	Spec  string
}

type QAFailInput struct {
	FeatureIssueID string
	Findings       []FindingInput
}

type QAFailResult struct {
	ParentIssueID string
	CreatedBugIDs []string
}

func (s *Service) QAFailWithFindings(ctx context.Context, input QAFailInput) (QAFailResult, error) {
	featureIssueID, err := shared.ParseID(input.FeatureIssueID)
	if err != nil {
		return QAFailResult{}, fmt.Errorf("invalid feature_issue_id: %w", err)
	}

	var result QAFailResult
	var parentProjOwner shared.ID

	err = s.uow.Execute(ctx, func(repos uow.Repositories) error {
		parentIssue, err := repos.Issues.FindByID(ctx, featureIssueID)
		if err != nil {
			return fmt.Errorf("failed to get parent issue: %w", err)
		}

		proj, err := repos.Projects.FindByID(ctx, parentIssue.ProjectID())
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}
		parentProjOwner = proj.OwnerID()

		// 1. Find or create the 'qa-finding' label
		labels, err := repos.Labels.FindByProject(ctx, parentIssue.ProjectID())
		if err != nil {
			return fmt.Errorf("failed to list project labels: %w", err)
		}

		var qaLabelID shared.ID
		for _, l := range labels {
			if strings.ToLower(l.Name().String()) == "qa-finding" {
				qaLabelID = l.ID()
				break
			}
		}

		if qaLabelID.IsZero() {
			name, _ := label.NewName("qa-finding")
			color, _ := label.NewColor("#D93F0B")
			newLabel, err := label.New(parentIssue.ProjectID(), name, color)
			if err != nil {
				return fmt.Errorf("failed to construct qa-finding label: %w", err)
			}
			if err := repos.Labels.Save(ctx, newLabel); err != nil {
				return fmt.Errorf("failed to save qa-finding label: %w", err)
			}
			qaLabelID = newLabel.ID()
		}

		var createdBugIDs []string
		for _, f := range input.Findings {
			title, err := issue.NewTitle(f.Title)
			if err != nil {
				return fmt.Errorf("invalid title for bug: %w", err)
			}

			newIss, err := issue.New(parentIssue.ProjectID(), parentIssue.PhaseID(), title)
			if err != nil {
				return fmt.Errorf("failed to create bug issue '%s': %w", f.Title, err)
			}
			newIss.SetSpec(f.Spec)
			newIss.SetType(issue.IssueTypeBug)
			newIss.SetPriority(issue.PriorityHigh)

			if err := repos.Issues.Save(ctx, newIss); err != nil {
				return fmt.Errorf("failed to save bug issue: %w", err)
			}

			if err := repos.Issues.AddLabel(ctx, newIss.ID(), qaLabelID); err != nil {
				// We can return error here since it's a transaction
				return fmt.Errorf("failed to add label to bug: %w", err)
			}

			createdBugIDs = append(createdBugIDs, newIss.ID().String())
		}

		// Comment
		commentBodyStr := fmt.Sprintf("QA Rejected. Created bugs: %v", createdBugIDs)
		cBody, err := comment.NewBody(commentBodyStr)
		if err != nil {
			return fmt.Errorf("invalid comment body: %w", err)
		}
		newComment, err := comment.New(featureIssueID, cBody)
		if err != nil {
			return fmt.Errorf("failed to create comment: %w", err)
		}
		if err := repos.Comments.Save(ctx, newComment); err != nil {
			return fmt.Errorf("failed to save comment: %w", err)
		}

		// Transition
		if err := parentIssue.Transition(issue.StatusInProgress); err != nil {
			return fmt.Errorf("failed to transition parent issue: %w", err)
		}
		if err := repos.Issues.Update(ctx, parentIssue); err != nil {
			return fmt.Errorf("failed to update parent issue: %w", err)
		}

		result.ParentIssueID = parentIssue.ID().String()
		result.CreatedBugIDs = createdBugIDs
		return nil
	})

	if err != nil {
		return QAFailResult{}, err
	}

	// Notify after successful transaction
	s.notifier.Notify(ctx, parentProjOwner, notification.Event{
		Event: "issue.updated",
		Payload: map[string]string{
			"id":         result.ParentIssueID,
			"event_type": "qa_fail",
		},
	})

	return result, nil
}

type FollowUpInput struct {
	Title string
	Spec  string
	Type  string
}

type ResolveInvestigationInput struct {
	IssueID        string
	Findings       string
	FollowUpIssues []FollowUpInput
}

type ResolveInvestigationResult struct {
	IssueID       string
	CreatedIssues []string
}

func (s *Service) ResolveInvestigation(ctx context.Context, input ResolveInvestigationInput) (ResolveInvestigationResult, error) {
	issueID, err := shared.ParseID(input.IssueID)
	if err != nil {
		return ResolveInvestigationResult{}, fmt.Errorf("invalid issue_id: %w", err)
	}

	var result ResolveInvestigationResult
	var parentProjOwner shared.ID

	err = s.uow.Execute(ctx, func(repos uow.Repositories) error {
		parentIssue, err := repos.Issues.FindByID(ctx, issueID)
		if err != nil {
			return fmt.Errorf("failed to get investigation issue: %w", err)
		}

		proj, err := repos.Projects.FindByID(ctx, parentIssue.ProjectID())
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}
		parentProjOwner = proj.OwnerID()

		var createdIssues []string
		for _, f := range input.FollowUpIssues {
			title, err := issue.NewTitle(f.Title)
			if err != nil {
				return fmt.Errorf("invalid title for follow-up: %w", err)
			}
			newIss, err := issue.New(parentIssue.ProjectID(), parentIssue.PhaseID(), title)
			if err != nil {
				return fmt.Errorf("failed to create follow-up issue: %w", err)
			}
			newIss.SetSpec(f.Spec)
			
			t := f.Type
			if t == "" {
				t = "task"
			}
			parsedType, err := issue.ParseIssueType(t)
			if err != nil {
				return fmt.Errorf("invalid type for follow-up: %w", err)
			}
			newIss.SetType(parsedType)

			if err := repos.Issues.Save(ctx, newIss); err != nil {
				return fmt.Errorf("failed to save follow-up issue: %w", err)
			}
			createdIssues = append(createdIssues, newIss.ID().String())
		}

		commentBodyStr := fmt.Sprintf("Investigation Findings:\n\n%s", input.Findings)
		if len(createdIssues) > 0 {
			commentBodyStr += fmt.Sprintf("\n\nFollow-up issues created: %v", createdIssues)
		}

		cBody, err := comment.NewBody(commentBodyStr)
		if err != nil {
			return fmt.Errorf("invalid comment body: %w", err)
		}
		newComment, err := comment.New(issueID, cBody)
		if err != nil {
			return fmt.Errorf("failed to create findings comment: %w", err)
		}
		if err := repos.Comments.Save(ctx, newComment); err != nil {
			return fmt.Errorf("failed to save findings comment: %w", err)
		}

		if err := parentIssue.Transition(issue.StatusDone); err != nil {
			return fmt.Errorf("failed to transition investigation issue: %w", err)
		}
		if err := repos.Issues.Update(ctx, parentIssue); err != nil {
			return fmt.Errorf("failed to update investigation issue: %w", err)
		}

		result.IssueID = parentIssue.ID().String()
		result.CreatedIssues = createdIssues
		return nil
	})

	if err != nil {
		return ResolveInvestigationResult{}, err
	}

	s.notifier.Notify(ctx, parentProjOwner, notification.Event{
		Event: "issue.updated",
		Payload: map[string]string{
			"id":         result.IssueID,
			"event_type": "investigation_resolved",
		},
	})

	return result, nil
}
