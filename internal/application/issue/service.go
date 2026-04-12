package issue

import (
	"context"
	"time"

	"project-management-tools/internal/application/notification"
	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

type CreateInput struct {
	ProjectID string
	PhaseID   *string // nil = backlog
	Title     string
	Spec      string
	Priority  string
	Type      string
	DueDate   *time.Time
}

type UpdateInput struct {
	Title        *string
	Spec         *string
	Priority     *string
	Type         *string
	DueDate      *time.Time // set to this value when non-nil
	ClearDueDate bool       // if true, clears the due date regardless of DueDate
}

type Service struct {
	repo        Repository
	phaseRepo   PhaseRepository
	projectRepo ProjectRepository
	labelRepo   LabelRepository
	notifier    notification.Notifier
}

func NewService(repo Repository, phaseRepo PhaseRepository, projectRepo ProjectRepository, labelRepo LabelRepository, notifier notification.Notifier) *Service {
	return &Service{repo: repo, phaseRepo: phaseRepo, projectRepo: projectRepo, labelRepo: labelRepo, notifier: notifier}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (issue.Issue, error) {
	projectID, err := shared.ParseID(input.ProjectID)
	if err != nil {
		return issue.Issue{}, err
	}

	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return issue.Issue{}, err
	}

	var phaseID *shared.ID
	if input.PhaseID != nil {
		pid, err := shared.ParseID(*input.PhaseID)
		if err != nil {
			return issue.Issue{}, err
		}
		if _, err := s.phaseRepo.FindByID(ctx, pid); err != nil {
			return issue.Issue{}, err
		}
		phaseID = &pid
	}

	title, err := issue.NewTitle(input.Title)
	if err != nil {
		return issue.Issue{}, err
	}

	iss, err := issue.New(projectID, phaseID, title)
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

	if input.Type != "" {
		issueType, err := issue.ParseIssueType(input.Type)
		if err != nil {
			return issue.Issue{}, err
		}
		iss.SetType(issueType)
	}

	if input.DueDate != nil {
		iss.SetDueDate(input.DueDate)
	}

	if err := s.repo.Save(ctx, iss); err != nil {
		return issue.Issue{}, err
	}

	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return issue.Issue{}, err
	}
	s.notifier.Notify(proj.OwnerID(), notification.Event{
		Event:   "issue.created",
		Payload: toNotificationPayload(iss),
	})

	return iss, nil
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (issue.Issue, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListByPhase(ctx context.Context, phaseID shared.ID) ([]issue.Issue, error) {
	return s.repo.FindByPhase(ctx, phaseID)
}

func (s *Service) ListBacklog(ctx context.Context, projectID shared.ID) ([]issue.Issue, error) {
	return s.repo.FindBacklog(ctx, projectID)
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

	if input.Type != nil {
		issueType, err := issue.ParseIssueType(*input.Type)
		if err != nil {
			return issue.Issue{}, err
		}
		iss.SetType(issueType)
	}

	if input.ClearDueDate {
		iss.SetDueDate(nil)
	} else if input.DueDate != nil {
		iss.SetDueDate(input.DueDate)
	}

	if err := s.repo.Update(ctx, iss); err != nil {
		return issue.Issue{}, err
	}

	proj, err := s.projectRepo.FindByID(ctx, iss.ProjectID())
	if err != nil {
		return issue.Issue{}, err
	}
	s.notifier.Notify(proj.OwnerID(), notification.Event{
		Event:   "issue.updated",
		Payload: toNotificationPayload(iss),
	})

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

	proj, err := s.projectRepo.FindByID(ctx, iss.ProjectID())
	if err != nil {
		return issue.Issue{}, err
	}
	s.notifier.Notify(proj.OwnerID(), notification.Event{
		Event:   "issue.updated",
		Payload: toNotificationPayload(iss),
	})

	return iss, nil
}

func (s *Service) Delete(ctx context.Context, id shared.ID) error {
	iss, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	proj, err := s.projectRepo.FindByID(ctx, iss.ProjectID())
	if err != nil {
		return err
	}
	s.notifier.Notify(proj.OwnerID(), notification.Event{
		Event:   "issue.deleted",
		Payload: map[string]string{"id": id.String(), "project_id": iss.ProjectID().String()},
	})
	return nil
}

func toNotificationPayload(i issue.Issue) map[string]string {
	var phaseID string
	if i.PhaseID() != nil {
		phaseID = i.PhaseID().String()
	}
	return map[string]string{
		"id":         i.ID().String(),
		"project_id": i.ProjectID().String(),
		"phase_id":   phaseID,
		"title":      i.Title().String(),
		"status":     string(i.Status()),
		"priority":   string(i.Priority()),
		"type":       string(i.Type()),
	}
}

func (s *Service) AddLabel(ctx context.Context, issueID, labelID shared.ID) error {
	if _, err := s.repo.FindByID(ctx, issueID); err != nil {
		return err
	}
	if _, err := s.labelRepo.FindByID(ctx, labelID); err != nil {
		return err
	}
	return s.repo.AddLabel(ctx, issueID, labelID)
}

func (s *Service) RemoveLabel(ctx context.Context, issueID, labelID shared.ID) error {
	return s.repo.RemoveLabel(ctx, issueID, labelID)
}
