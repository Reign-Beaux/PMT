package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/shared"
)

type issueModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	PhaseID   string    `gorm:"not null;type:uuid;index"`
	Title     string    `gorm:"not null"`
	Spec      string    `gorm:"not null;default:''"`
	Status    string    `gorm:"not null;default:'open'"`
	Priority  string    `gorm:"not null;default:'medium'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (issueModel) TableName() string { return "issues" }

type IssueRepository struct {
	db *gorm.DB
}

func NewIssueRepository(db *gorm.DB) *IssueRepository {
	return &IssueRepository{db: db}
}

func (r *IssueRepository) Save(ctx context.Context, i issue.Issue) error {
	model := toIssueModel(i)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *IssueRepository) FindByID(ctx context.Context, id shared.ID) (issue.Issue, error) {
	var model issueModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return issue.Issue{}, issue.ErrNotFound
	}
	if err != nil {
		return issue.Issue{}, err
	}
	return toIssueDomain(model)
}

func (r *IssueRepository) FindByPhase(ctx context.Context, phaseID shared.ID) ([]issue.Issue, error) {
	var models []issueModel
	err := r.db.WithContext(ctx).
		Where("phase_id = ?", phaseID.String()).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	issues := make([]issue.Issue, 0, len(models))
	for _, m := range models {
		iss, err := toIssueDomain(m)
		if err != nil {
			return nil, err
		}
		issues = append(issues, iss)
	}
	return issues, nil
}

func (r *IssueRepository) Update(ctx context.Context, i issue.Issue) error {
	model := toIssueModel(i)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *IssueRepository) Delete(ctx context.Context, id shared.ID) error {
	return r.db.WithContext(ctx).Delete(&issueModel{}, "id = ?", id.String()).Error
}

func toIssueModel(i issue.Issue) issueModel {
	return issueModel{
		ID:        i.ID().String(),
		PhaseID:   i.PhaseID().String(),
		Title:     i.Title().String(),
		Spec:      i.Spec(),
		Status:    string(i.Status()),
		Priority:  string(i.Priority()),
		CreatedAt: i.CreatedAt(),
		UpdatedAt: i.UpdatedAt(),
	}
}

func toIssueDomain(m issueModel) (issue.Issue, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return issue.Issue{}, err
	}
	phaseID, err := shared.ParseID(m.PhaseID)
	if err != nil {
		return issue.Issue{}, err
	}
	title, err := issue.NewTitle(m.Title)
	if err != nil {
		return issue.Issue{}, err
	}
	status, err := issue.ParseStatus(m.Status)
	if err != nil {
		return issue.Issue{}, err
	}
	priority, err := issue.ParsePriority(m.Priority)
	if err != nil {
		return issue.Issue{}, err
	}
	return issue.Reconstitute(id, phaseID, title, m.Spec, status, priority, m.CreatedAt, m.UpdatedAt), nil
}
