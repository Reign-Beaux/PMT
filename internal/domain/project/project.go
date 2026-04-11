package project

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type Project struct {
	id          shared.ID
	name        Name
	description string
	status      Status
	createdAt   time.Time
	updatedAt   time.Time
}

func New(name Name) (Project, error) {
	if !name.isValid() {
		return Project{}, ErrInvalidName
	}
	now := time.Now()
	return Project{
		id:        shared.NewID(),
		name:      name,
		status:    StatusActive,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute rebuilds a Project from persisted data.
// Bypasses constructor validation — callers must ensure data integrity.
func Reconstitute(id shared.ID, name Name, description string, status Status, createdAt, updatedAt time.Time) Project {
	return Project{
		id:          id,
		name:        name,
		description: description,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

func (p Project) ID() shared.ID       { return p.id }
func (p Project) Name() Name          { return p.name }
func (p Project) Description() string { return p.description }
func (p Project) Status() Status      { return p.status }
func (p Project) CreatedAt() time.Time { return p.createdAt }
func (p Project) UpdatedAt() time.Time { return p.updatedAt }

func (p *Project) Rename(name Name) error {
	if !name.isValid() {
		return ErrInvalidName
	}
	p.name = name
	p.updatedAt = time.Now()
	return nil
}

func (p *Project) SetDescription(desc string) {
	p.description = desc
	p.updatedAt = time.Now()
}

func (p *Project) ChangeStatus(s Status) error {
	_, err := ParseStatus(string(s))
	if err != nil {
		return err
	}
	p.status = s
	p.updatedAt = time.Now()
	return nil
}

func (p *Project) Archive() {
	p.status = StatusArchived
	p.updatedAt = time.Now()
}
