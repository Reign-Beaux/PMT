package comment

import (
	"context"

	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/shared"
)

type CreateInput struct {
	IssueID string
	Body    string
}

type Service struct {
	repo      Repository
	issueRepo IssueRepository
}

func NewService(repo Repository, issueRepo IssueRepository) *Service {
	return &Service{repo: repo, issueRepo: issueRepo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (comment.Comment, error) {
	issueID, err := shared.ParseID(input.IssueID)
	if err != nil {
		return comment.Comment{}, err
	}

	if _, err := s.issueRepo.FindByID(ctx, issueID); err != nil {
		return comment.Comment{}, err
	}

	body, err := comment.NewBody(input.Body)
	if err != nil {
		return comment.Comment{}, err
	}

	c, err := comment.New(issueID, body)
	if err != nil {
		return comment.Comment{}, err
	}

	if err := s.repo.Save(ctx, c); err != nil {
		return comment.Comment{}, err
	}

	return c, nil
}

func (s *Service) ListByIssue(ctx context.Context, issueID shared.ID) ([]comment.Comment, error) {
	return s.repo.FindByIssue(ctx, issueID)
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
