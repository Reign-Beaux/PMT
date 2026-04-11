package project

import (
	"context"

	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

type CreateInput struct {
	OwnerID     shared.ID
	Name        string
	Description string
}

type UpdateInput struct {
	Name        *string
	Description *string
	Status      *project.Status
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (project.Project, error) {
	name, err := project.NewName(input.Name)
	if err != nil {
		return project.Project{}, err
	}

	p, err := project.New(input.OwnerID, name)
	if err != nil {
		return project.Project{}, err
	}

	if input.Description != "" {
		p.SetDescription(input.Description)
	}

	if err := s.repo.Save(ctx, p); err != nil {
		return project.Project{}, err
	}

	return p, nil
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (project.Project, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, ownerID shared.ID) ([]project.Project, error) {
	return s.repo.FindByOwnerID(ctx, ownerID)
}

func (s *Service) Update(ctx context.Context, id shared.ID, input UpdateInput) (project.Project, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return project.Project{}, err
	}

	if input.Name != nil {
		name, err := project.NewName(*input.Name)
		if err != nil {
			return project.Project{}, err
		}
		if err := p.Rename(name); err != nil {
			return project.Project{}, err
		}
	}

	if input.Description != nil {
		p.SetDescription(*input.Description)
	}

	if input.Status != nil {
		if err := p.ChangeStatus(*input.Status); err != nil {
			return project.Project{}, err
		}
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return project.Project{}, err
	}

	return p, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
