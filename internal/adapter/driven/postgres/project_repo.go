package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

type projectModel struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID      string    `gorm:"not null;index"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"not null;default:''"`
	Status      string    `gorm:"not null;default:'active'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (projectModel) TableName() string { return "projects" }

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Save(ctx context.Context, p project.Project) error {
	model := toProjectModel(p)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *ProjectRepository) FindByID(ctx context.Context, id shared.ID) (project.Project, error) {
	var model projectModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return project.Project{}, project.ErrNotFound
	}
	if err != nil {
		return project.Project{}, err
	}
	return toProjectDomain(model)
}

func (r *ProjectRepository) FindByOwnerID(ctx context.Context, ownerID shared.ID) ([]project.Project, error) {
	var models []projectModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", ownerID.String()).Find(&models).Error; err != nil {
		return nil, err
	}

	projects := make([]project.Project, 0, len(models))
	for _, m := range models {
		p, err := toProjectDomain(m)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *ProjectRepository) Update(ctx context.Context, p project.Project) error {
	model := toProjectModel(p)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *ProjectRepository) Delete(ctx context.Context, id shared.ID) error {
	return r.db.WithContext(ctx).Delete(&projectModel{}, "id = ?", id.String()).Error
}

func toProjectModel(p project.Project) projectModel {
	return projectModel{
		ID:          p.ID().String(),
		UserID:      p.OwnerID().String(),
		Name:        p.Name().String(),
		Description: p.Description(),
		Status:      string(p.Status()),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

func toProjectDomain(m projectModel) (project.Project, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return project.Project{}, err
	}
	ownerID, err := shared.ParseID(m.UserID)
	if err != nil {
		return project.Project{}, err
	}
	name, err := project.NewName(m.Name)
	if err != nil {
		return project.Project{}, err
	}
	status, err := project.ParseStatus(m.Status)
	if err != nil {
		return project.Project{}, err
	}
	return project.Reconstitute(id, ownerID, name, m.Description, status, m.CreatedAt, m.UpdatedAt), nil
}
