package issue

import (
	"context"

	"project-management-tools/internal/domain/issue"
	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/shared"
)

// Repository is the driven port for issue persistence.
type Repository interface {
	Save(ctx context.Context, i issue.Issue) error
	FindByID(ctx context.Context, id shared.ID) (issue.Issue, error)
	FindByPhase(ctx context.Context, phaseID shared.ID) ([]issue.Issue, error)
	Update(ctx context.Context, i issue.Issue) error
	Delete(ctx context.Context, id shared.ID) error
}

// PhaseRepository allows the issue service to verify phase existence.
// Defined here, in the consumer.
type PhaseRepository interface {
	FindByID(ctx context.Context, id shared.ID) (phase.Phase, error)
}
