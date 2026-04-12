package phase

import (
	"context"
	"fmt"

	"project-management-tools/internal/application/notification"
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
	notifier    notification.Notifier
}

func NewService(repo Repository, projectRepo ProjectRepository, notifier notification.Notifier) *Service {
	return &Service{repo: repo, projectRepo: projectRepo, notifier: notifier}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (phase.Phase, error) {
	projectID, err := shared.ParseID(input.ProjectID)
	if err != nil {
		return phase.Phase{}, err
	}

	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
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

	s.notifier.Notify(ctx, proj.OwnerID(), notification.Event{
		Event:   "phase.created",
		Payload: toNotificationPayload(p),
	})

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

	proj, err := s.projectRepo.FindByID(ctx, p.ProjectID())
	if err != nil {
		return phase.Phase{}, err
	}
	s.notifier.Notify(ctx, proj.OwnerID(), notification.Event{
		Event:   "phase.updated",
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
	proj, err := s.projectRepo.FindByID(ctx, p.ProjectID())
	if err != nil {
		return err
	}
	s.notifier.Notify(ctx, proj.OwnerID(), notification.Event{
		Event:   "phase.deleted",
		Payload: map[string]string{"id": id.String(), "project_id": p.ProjectID().String()},
	})
	return nil
}

func toNotificationPayload(p phase.Phase) map[string]string {
	return map[string]string{
		"id":          p.ID().String(),
		"project_id":   p.ProjectID().String(),
		"name":        p.Name().String(),
		"description": p.Description(),
		"order":       fmt.Sprintf("%d", p.Order().Value()),
		"status":      string(p.Status()),
	}
}
