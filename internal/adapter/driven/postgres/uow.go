package postgres

import (
	"context"

	"gorm.io/gorm"

	"project-management-tools/internal/application/uow"
)

// UnitOfWork coordinates multi-aggregate operations within a single database transaction.
//
// Use cases that touch a single aggregate inject their repository directly — this struct
// is only for operations that require atomicity across multiple aggregates.
//
// The driven port interface is defined in the application use case that consumes it,
// not here. Each consuming use case declares:
//
//	type unitOfWork interface {
//	    Execute(ctx context.Context, fn func(uow.Repositories) error) error
//	}
type UnitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

// Execute runs fn inside a database transaction.
// GORM commits automatically when fn returns nil, and rolls back on any error.
// The error propagates up to the use case, then to the HTTP handler.
func (u *UnitOfWork) Execute(ctx context.Context, fn func(repos uow.Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(uow.Repositories{
			Projects: NewProjectRepository(tx),
			Phases:   NewPhaseRepository(tx),
			Issues:   NewIssueRepository(tx),
		})
	})
}
