package project

import (
	"context"

	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

// Repository is the driven port for project persistence.
// Defined here, in the consumer (application layer).
type Repository interface {
	Save(ctx context.Context, p project.Project) error
	FindByID(ctx context.Context, id shared.ID) (project.Project, error)
	FindByOwnerID(ctx context.Context, ownerID shared.ID) ([]project.Project, error)
	Update(ctx context.Context, p project.Project) error
	Delete(ctx context.Context, id shared.ID) error
}
