package comment

import (
	"context"

	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

// Repository is the driven port for comment persistence.
type Repository interface {
	Save(ctx context.Context, c comment.Comment) error
	FindByID(ctx context.Context, id shared.ID) (comment.Comment, error)
	FindByIssue(ctx context.Context, issueID shared.ID) ([]comment.Comment, error)
	Delete(ctx context.Context, id shared.ID) error
}

// IssueRepository allows the comment service to verify issue existence.
// Defined here, in the consumer.
type IssueRepository interface {
	FindByID(ctx context.Context, id shared.ID) (issue.Issue, error)
}
