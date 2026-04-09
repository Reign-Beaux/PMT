package label

import (
	"context"

	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// Repository is the driven port for label persistence.
type Repository interface {
	Save(ctx context.Context, l label.Label) error
	FindByID(ctx context.Context, id shared.ID) (label.Label, error)
	FindByProject(ctx context.Context, projectID shared.ID) ([]label.Label, error)
	Update(ctx context.Context, l label.Label) error
	Delete(ctx context.Context, id shared.ID) error
}

// ProjectRepository allows the label service to verify project existence.
// Defined here, in the consumer.
type ProjectRepository interface {
	FindByID(ctx context.Context, id shared.ID) (project.Project, error)
}
