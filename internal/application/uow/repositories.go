// Package uow provides shared types for coordinating multi-aggregate transactions.
// It is not an aggregate — it is infrastructure for the application layer.
package uow

import (
	"project-management-tools/internal/application/issue"
	"project-management-tools/internal/application/phase"
	"project-management-tools/internal/application/project"
)

// Repositories groups all aggregate repositories available within a transaction.
// It is the argument passed to the callback in UnitOfWork.Execute.
// Defined here (application layer) so the driven port interface can reference it
// without importing the postgres adapter.
type Repositories struct {
	Projects project.Repository
	Phases   phase.Repository
	Issues   issue.Repository
}
