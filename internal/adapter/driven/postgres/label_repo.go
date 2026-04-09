package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/shared"
)

type labelModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ProjectID string    `gorm:"not null;type:uuid;index"`
	Name      string    `gorm:"not null"`
	Color     string    `gorm:"not null;default:'#6366f1'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (labelModel) TableName() string { return "labels" }

type LabelRepository struct {
	db *gorm.DB
}

func NewLabelRepository(db *gorm.DB) *LabelRepository {
	return &LabelRepository{db: db}
}

func (r *LabelRepository) Save(ctx context.Context, l label.Label) error {
	model := toLabelModel(l)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *LabelRepository) FindByID(ctx context.Context, id shared.ID) (label.Label, error) {
	var model labelModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return label.Label{}, label.ErrNotFound
	}
	if err != nil {
		return label.Label{}, err
	}
	return toLabelDomain(model)
}

func (r *LabelRepository) FindByProject(ctx context.Context, projectID shared.ID) ([]label.Label, error) {
	var models []labelModel
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Order("name ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	labels := make([]label.Label, 0, len(models))
	for _, m := range models {
		l, err := toLabelDomain(m)
		if err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, nil
}

func (r *LabelRepository) Update(ctx context.Context, l label.Label) error {
	model := toLabelModel(l)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *LabelRepository) Delete(ctx context.Context, id shared.ID) error {
	return r.db.WithContext(ctx).Delete(&labelModel{}, "id = ?", id.String()).Error
}

func toLabelModel(l label.Label) labelModel {
	return labelModel{
		ID:        l.ID().String(),
		ProjectID: l.ProjectID().String(),
		Name:      l.Name().String(),
		Color:     l.Color().String(),
		CreatedAt: l.CreatedAt(),
		UpdatedAt: l.UpdatedAt(),
	}
}

func toLabelDomain(m labelModel) (label.Label, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return label.Label{}, err
	}
	projectID, err := shared.ParseID(m.ProjectID)
	if err != nil {
		return label.Label{}, err
	}
	name, err := label.NewName(m.Name)
	if err != nil {
		return label.Label{}, err
	}
	color, err := label.NewColor(m.Color)
	if err != nil {
		return label.Label{}, err
	}
	return label.Reconstitute(id, projectID, name, color, m.CreatedAt, m.UpdatedAt), nil
}
