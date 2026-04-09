package label

import (
	"context"

	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/shared"
)

type CreateInput struct {
	ProjectID string
	Name      string
	Color     string
}

type UpdateInput struct {
	Name  *string
	Color *string
}

type Service struct {
	repo        Repository
	projectRepo ProjectRepository
}

func NewService(repo Repository, projectRepo ProjectRepository) *Service {
	return &Service{repo: repo, projectRepo: projectRepo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (label.Label, error) {
	projectID, err := shared.ParseID(input.ProjectID)
	if err != nil {
		return label.Label{}, err
	}

	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return label.Label{}, err
	}

	name, err := label.NewName(input.Name)
	if err != nil {
		return label.Label{}, err
	}

	color, err := label.NewColor(input.Color)
	if err != nil {
		return label.Label{}, err
	}

	l, err := label.New(projectID, name, color)
	if err != nil {
		return label.Label{}, err
	}

	if err := s.repo.Save(ctx, l); err != nil {
		return label.Label{}, err
	}

	return l, nil
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (label.Label, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListByProject(ctx context.Context, projectID shared.ID) ([]label.Label, error) {
	return s.repo.FindByProject(ctx, projectID)
}

func (s *Service) Update(ctx context.Context, id shared.ID, input UpdateInput) (label.Label, error) {
	l, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return label.Label{}, err
	}

	if input.Name != nil {
		name, err := label.NewName(*input.Name)
		if err != nil {
			return label.Label{}, err
		}
		if err := l.Rename(name); err != nil {
			return label.Label{}, err
		}
	}

	if input.Color != nil {
		color, err := label.NewColor(*input.Color)
		if err != nil {
			return label.Label{}, err
		}
		l.SetColor(color)
	}

	if err := s.repo.Update(ctx, l); err != nil {
		return label.Label{}, err
	}

	return l, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
