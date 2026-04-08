package phase

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type Phase struct {
	id          shared.ID
	projectID   shared.ID
	name        Name
	description string
	order       Order
	status      Status
	createdAt   time.Time
	updatedAt   time.Time
}

func New(projectID shared.ID, name Name, order Order) (Phase, error) {
	if projectID.IsZero() {
		return Phase{}, ErrInvalidProjectID
	}
	if !name.isValid() {
		return Phase{}, ErrInvalidName
	}
	if !order.isValid() {
		return Phase{}, ErrInvalidOrder
	}
	now := time.Now()
	return Phase{
		id:        shared.NewID(),
		projectID: projectID,
		name:      name,
		order:     order,
		status:    StatusActive,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute rebuilds a Phase from persisted data.
// Bypasses constructor validation — callers must ensure data integrity.
func Reconstitute(id, projectID shared.ID, name Name, description string, order Order, status Status, createdAt, updatedAt time.Time) Phase {
	return Phase{
		id:          id,
		projectID:   projectID,
		name:        name,
		description: description,
		order:       order,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

func (p Phase) ID() shared.ID        { return p.id }
func (p Phase) ProjectID() shared.ID { return p.projectID }
func (p Phase) Name() Name           { return p.name }
func (p Phase) Description() string  { return p.description }
func (p Phase) Order() Order         { return p.order }
func (p Phase) Status() Status       { return p.status }
func (p Phase) CreatedAt() time.Time { return p.createdAt }
func (p Phase) UpdatedAt() time.Time { return p.updatedAt }

func (p *Phase) Rename(name Name) error {
	if !name.isValid() {
		return ErrInvalidName
	}
	p.name = name
	p.updatedAt = time.Now()
	return nil
}

func (p *Phase) SetDescription(desc string) {
	p.description = desc
	p.updatedAt = time.Now()
}

func (p *Phase) Complete() {
	p.status = StatusCompleted
	p.updatedAt = time.Now()
}
