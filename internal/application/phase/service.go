package phase

import (
	"context"

	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/shared"
)

type CreateInput struct {
	ProjectID   string
	Name        string
	Description string
}

type UpdateInput struct {
	Name        *string
	Description *string
}

type Service struct {
	repo        Repository
	projectRepo ProjectRepository
}

func NewService(repo Repository, projectRepo ProjectRepository) *Service {
	return &Service{repo: repo, projectRepo: projectRepo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (phase.Phase, error) {
	projectID, err := shared.ParseID(input.ProjectID)
	if err != nil {
		return phase.Phase{}, err
	}

	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return phase.Phase{}, err
	}

	name, err := phase.NewName(input.Name)
	if err != nil {
		return phase.Phase{}, err
	}

	count, err := s.repo.CountByProject(ctx, projectID)
	if err != nil {
		return phase.Phase{}, err
	}

	order, err := phase.NewOrder(count + 1)
	if err != nil {
		return phase.Phase{}, err
	}

	p, err := phase.New(projectID, name, order)
	if err != nil {
		return phase.Phase{}, err
	}

	if input.Description != "" {
		p.SetDescription(input.Description)
	}

	if err := s.repo.Save(ctx, p); err != nil {
		return phase.Phase{}, err
	}

	return p, nil
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (phase.Phase, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListByProject(ctx context.Context, projectID shared.ID) ([]phase.Phase, error) {
	return s.repo.FindByProject(ctx, projectID)
}

func (s *Service) Update(ctx context.Context, id shared.ID, input UpdateInput) (phase.Phase, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return phase.Phase{}, err
	}

	if input.Name != nil {
		name, err := phase.NewName(*input.Name)
		if err != nil {
			return phase.Phase{}, err
		}
		if err := p.Rename(name); err != nil {
			return phase.Phase{}, err
		}
	}

	if input.Description != nil {
		p.SetDescription(*input.Description)
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return phase.Phase{}, err
	}

	return p, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
