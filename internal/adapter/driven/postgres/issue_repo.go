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
	ID        string       `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ProjectID string       `gorm:"not null;type:uuid;index"`
	PhaseID   *string      `gorm:"type:uuid;index"`
	Type      string       `gorm:"column:type;not null;default:'task'"`
	Title     string       `gorm:"not null"`
	Spec      string       `gorm:"not null;default:''"`
	Status    string       `gorm:"not null;default:'open'"`
	Priority  string       `gorm:"not null;default:'medium'"`
	DueDate   *time.Time   `gorm:"type:timestamptz"`
	Labels    []labelModel `gorm:"many2many:issue_labels;"`
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
	err := r.db.WithContext(ctx).
		Preload("Labels").
		First(&model, "id = ?", id.String()).Error
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
		Preload("Labels").
		Where("phase_id = ?", phaseID.String()).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	return mapIssueModels(models)
}

func (r *IssueRepository) FindBacklog(ctx context.Context, projectID shared.ID) ([]issue.Issue, error) {
	var models []issueModel
	err := r.db.WithContext(ctx).
		Preload("Labels").
		Where("project_id = ? AND phase_id IS NULL", projectID.String()).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	return mapIssueModels(models)
}

func (r *IssueRepository) Update(ctx context.Context, i issue.Issue) error {
	model := toIssueModel(i)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *IssueRepository) Delete(ctx context.Context, id shared.ID) error {
	return r.db.WithContext(ctx).Delete(&issueModel{}, "id = ?", id.String()).Error
}

func (r *IssueRepository) AddLabel(ctx context.Context, issueID, labelID shared.ID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO issue_labels (issue_id, label_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		issueID.String(), labelID.String(),
	).Error
}

func (r *IssueRepository) RemoveLabel(ctx context.Context, issueID, labelID shared.ID) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM issue_labels WHERE issue_id = ? AND label_id = ?",
		issueID.String(), labelID.String(),
	).Error
}

func toIssueModel(i issue.Issue) issueModel {
	var phaseID *string
	if i.PhaseID() != nil {
		s := i.PhaseID().String()
		phaseID = &s
	}
	return issueModel{
		ID:        i.ID().String(),
		ProjectID: i.ProjectID().String(),
		PhaseID:   phaseID,
		Type:      string(i.Type()),
		Title:     i.Title().String(),
		Spec:      i.Spec(),
		Status:    string(i.Status()),
		Priority:  string(i.Priority()),
		DueDate:   i.DueDate(),
		CreatedAt: i.CreatedAt(),
		UpdatedAt: i.UpdatedAt(),
	}
}

func toIssueDomain(m issueModel) (issue.Issue, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return issue.Issue{}, err
	}
	projectID, err := shared.ParseID(m.ProjectID)
	if err != nil {
		return issue.Issue{}, err
	}
	var phaseID *shared.ID
	if m.PhaseID != nil {
		pid, err := shared.ParseID(*m.PhaseID)
		if err != nil {
			return issue.Issue{}, err
		}
		phaseID = &pid
	}
	issueType, err := issue.ParseIssueType(m.Type)
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
	labelIDs := make([]shared.ID, 0, len(m.Labels))
	for _, lm := range m.Labels {
		lid, err := shared.ParseID(lm.ID)
		if err != nil {
			return issue.Issue{}, err
		}
		labelIDs = append(labelIDs, lid)
	}
	return issue.Reconstitute(id, projectID, phaseID, issueType, title, m.Spec, status, priority, m.DueDate, labelIDs, m.CreatedAt, m.UpdatedAt), nil
}

func mapIssueModels(models []issueModel) ([]issue.Issue, error) {
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
