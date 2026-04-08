package issue

import (
	"context"

	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

type CreateInput struct {
	PhaseID  string
	Title    string
	Spec     string
	Priority string
}

type UpdateInput struct {
	Title    *string
	Spec     *string
	Priority *string
}

type Service struct {
	repo      Repository
	phaseRepo PhaseRepository
}

func NewService(repo Repository, phaseRepo PhaseRepository) *Service {
	return &Service{repo: repo, phaseRepo: phaseRepo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (issue.Issue, error) {
	phaseID, err := shared.ParseID(input.PhaseID)
	if err != nil {
		return issue.Issue{}, err
	}

	if _, err := s.phaseRepo.FindByID(ctx, phaseID); err != nil {
		return issue.Issue{}, err
	}

	title, err := issue.NewTitle(input.Title)
	if err != nil {
		return issue.Issue{}, err
	}

	iss, err := issue.New(phaseID, title)
	if err != nil {
		return issue.Issue{}, err
	}

	if input.Spec != "" {
		iss.SetSpec(input.Spec)
	}

	if input.Priority != "" {
		priority, err := issue.ParsePriority(input.Priority)
		if err != nil {
			return issue.Issue{}, err
		}
		iss.SetPriority(priority)
	}

	if err := s.repo.Save(ctx, iss); err != nil {
		return issue.Issue{}, err
	}

	return iss, nil
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (issue.Issue, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListByPhase(ctx context.Context, phaseID shared.ID) ([]issue.Issue, error) {
	return s.repo.FindByPhase(ctx, phaseID)
}

func (s *Service) Update(ctx context.Context, id shared.ID, input UpdateInput) (issue.Issue, error) {
	iss, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return issue.Issue{}, err
	}

	if input.Title != nil {
		title, err := issue.NewTitle(*input.Title)
		if err != nil {
			return issue.Issue{}, err
		}
		if err := iss.UpdateTitle(title); err != nil {
			return issue.Issue{}, err
		}
	}

	if input.Spec != nil {
		iss.SetSpec(*input.Spec)
	}

	if input.Priority != nil {
		priority, err := issue.ParsePriority(*input.Priority)
		if err != nil {
			return issue.Issue{}, err
		}
		iss.SetPriority(priority)
	}

	if err := s.repo.Update(ctx, iss); err != nil {
		return issue.Issue{}, err
	}

	return iss, nil
}

func (s *Service) Transition(ctx context.Context, id shared.ID, status string) (issue.Issue, error) {
	iss, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return issue.Issue{}, err
	}

	next, err := issue.ParseStatus(status)
	if err != nil {
		return issue.Issue{}, err
	}

	if err := iss.Transition(next); err != nil {
		return issue.Issue{}, err
	}

	if err := s.repo.Update(ctx, iss); err != nil {
		return issue.Issue{}, err
	}

	return iss, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
