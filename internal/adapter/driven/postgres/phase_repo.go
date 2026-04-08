package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/shared"
)

type phaseModel struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ProjectID   string    `gorm:"not null;type:uuid;index"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"not null;default:''"`
	SortOrder   int       `gorm:"column:sort_order;not null"`
	Status      string    `gorm:"not null;default:'active'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (phaseModel) TableName() string { return "phases" }

type PhaseRepository struct {
	db *gorm.DB
}

func NewPhaseRepository(db *gorm.DB) *PhaseRepository {
	return &PhaseRepository{db: db}
}

func (r *PhaseRepository) Save(ctx context.Context, p phase.Phase) error {
	model := toPhaseModel(p)
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *PhaseRepository) FindByID(ctx context.Context, id shared.ID) (phase.Phase, error) {
	var model phaseModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return phase.Phase{}, phase.ErrNotFound
	}
	if err != nil {
		return phase.Phase{}, err
	}
	return toPhaseDomain(model)
}

func (r *PhaseRepository) FindByProject(ctx context.Context, projectID shared.ID) ([]phase.Phase, error) {
	var models []phaseModel
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID.String()).
		Order("sort_order ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	phases := make([]phase.Phase, 0, len(models))
	for _, m := range models {
		p, err := toPhaseDomain(m)
		if err != nil {
			return nil, err
		}
		phases = append(phases, p)
	}
	return phases, nil
}

func (r *PhaseRepository) CountByProject(ctx context.Context, projectID shared.ID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&phaseModel{}).
		Where("project_id = ?", projectID.String()).
		Count(&count).Error
	return int(count), err
}

func (r *PhaseRepository) Update(ctx context.Context, p phase.Phase) error {
	model := toPhaseModel(p)
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *PhaseRepository) Delete(ctx context.Context, id shared.ID) error {
	return r.db.WithContext(ctx).Delete(&phaseModel{}, "id = ?", id.String()).Error
}

func toPhaseModel(p phase.Phase) phaseModel {
	return phaseModel{
		ID:          p.ID().String(),
		ProjectID:   p.ProjectID().String(),
		Name:        p.Name().String(),
		Description: p.Description(),
		SortOrder:   p.Order().Value(),
		Status:      string(p.Status()),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

func toPhaseDomain(m phaseModel) (phase.Phase, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return phase.Phase{}, err
	}
	projectID, err := shared.ParseID(m.ProjectID)
	if err != nil {
		return phase.Phase{}, err
	}
	name, err := phase.NewName(m.Name)
	if err != nil {
		return phase.Phase{}, err
	}
	order, err := phase.NewOrder(m.SortOrder)
	if err != nil {
		return phase.Phase{}, err
	}
	status, err := phase.ParseStatus(m.Status)
	if err != nil {
		return phase.Phase{}, err
	}
	return phase.Reconstitute(id, projectID, name, m.Description, order, status, m.CreatedAt, m.UpdatedAt), nil
}
