package phase

import (
	"context"

	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// Repository is the driven port for phase persistence.
type Repository interface {
	Save(ctx context.Context, p phase.Phase) error
	FindByID(ctx context.Context, id shared.ID) (phase.Phase, error)
	FindByProject(ctx context.Context, projectID shared.ID) ([]phase.Phase, error)
	CountByProject(ctx context.Context, projectID shared.ID) (int, error)
	Update(ctx context.Context, p phase.Phase) error
	Delete(ctx context.Context, id shared.ID) error
}

// ProjectRepository allows the phase service to verify project existence.
// Defined here, in the consumer.
type ProjectRepository interface {
	FindByID(ctx context.Context, id shared.ID) (project.Project, error)
}
