package label

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type Label struct {
	id        shared.ID
	projectID shared.ID
	name      Name
	color     Color
	createdAt time.Time
	updatedAt time.Time
}

func New(projectID shared.ID, name Name, color Color) (Label, error) {
	if projectID.IsZero() {
		return Label{}, ErrInvalidProjectID
	}
	if !name.isValid() {
		return Label{}, ErrInvalidName
	}
	now := time.Now()
	return Label{
		id:        shared.NewID(),
		projectID: projectID,
		name:      name,
		color:     color,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute rebuilds a Label from persisted data.
func Reconstitute(id, projectID shared.ID, name Name, color Color, createdAt, updatedAt time.Time) Label {
	return Label{
		id:        id,
		projectID: projectID,
		name:      name,
		color:     color,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (l Label) ID() shared.ID        { return l.id }
func (l Label) ProjectID() shared.ID { return l.projectID }
func (l Label) Name() Name           { return l.name }
func (l Label) Color() Color         { return l.color }
func (l Label) CreatedAt() time.Time { return l.createdAt }
func (l Label) UpdatedAt() time.Time { return l.updatedAt }

func (l *Label) Rename(name Name) error {
	if !name.isValid() {
		return ErrInvalidName
	}
	l.name = name
	l.updatedAt = time.Now()
	return nil
}

func (l *Label) SetColor(color Color) {
	l.color = color
	l.updatedAt = time.Now()
}
