package project

import (
	"context"

	"project-management-tools/internal/application/notification"
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
	repo     Repository
	notifier notification.Notifier
}

func NewService(repo Repository, notifier notification.Notifier) *Service {
	return &Service{repo: repo, notifier: notifier}
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

	s.notifier.Notify(p.OwnerID(), notification.Event{
		Event:   "project.created",
		Payload: toNotificationPayload(p),
	})

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

	s.notifier.Notify(p.OwnerID(), notification.Event{
		Event:   "project.updated",
		Payload: toNotificationPayload(p),
	})

	return p, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.notifier.Notify(p.OwnerID(), notification.Event{
		Event:   "project.deleted",
		Payload: map[string]string{"id": id.String()},
	})
	return nil
}

func toNotificationPayload(p project.Project) map[string]string {
	return map[string]string{
		"id":          p.ID().String(),
		"owner_id":     p.OwnerID().String(),
		"name":        p.Name().String(),
		"description": p.Description(),
		"status":      string(p.Status()),
	}
}
