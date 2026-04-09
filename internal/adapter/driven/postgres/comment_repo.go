package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"project-management-tools/internal/domain/comment"
	"project-management-tools/internal/domain/shared"
)

type commentModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	IssueID   string    `gorm:"not null;type:uuid;index"`
	Body      string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (commentModel) TableName() string { return "comments" }

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Save(ctx context.Context, c comment.Comment) error {
	model := toCommentModel(c)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *CommentRepository) FindByID(ctx context.Context, id shared.ID) (comment.Comment, error) {
	var model commentModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comment.Comment{}, comment.ErrNotFound
	}
	if err != nil {
		return comment.Comment{}, err
	}
	return toCommentDomain(model)
}

func (r *CommentRepository) FindByIssue(ctx context.Context, issueID shared.ID) ([]comment.Comment, error) {
	var models []commentModel
	err := r.db.WithContext(ctx).
		Where("issue_id = ?", issueID.String()).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	comments := make([]comment.Comment, 0, len(models))
	for _, m := range models {
		c, err := toCommentDomain(m)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *CommentRepository) Delete(ctx context.Context, id shared.ID) error {
	return r.db.WithContext(ctx).Delete(&commentModel{}, "id = ?", id.String()).Error
}

func toCommentModel(c comment.Comment) commentModel {
	return commentModel{
		ID:        c.ID().String(),
		IssueID:   c.IssueID().String(),
		Body:      c.Body().String(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

func toCommentDomain(m commentModel) (comment.Comment, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return comment.Comment{}, err
	}
	issueID, err := shared.ParseID(m.IssueID)
	if err != nil {
		return comment.Comment{}, err
	}
	body, err := comment.NewBody(m.Body)
	if err != nil {
		return comment.Comment{}, err
	}
	return comment.Reconstitute(id, issueID, body, m.CreatedAt, m.UpdatedAt), nil
}
