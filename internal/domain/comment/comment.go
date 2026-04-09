package comment

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type Comment struct {
	id        shared.ID
	issueID   shared.ID
	body      Body
	createdAt time.Time
	updatedAt time.Time
}

func New(issueID shared.ID, body Body) (Comment, error) {
	if issueID.IsZero() {
		return Comment{}, ErrInvalidBody
	}
	if !body.isValid() {
		return Comment{}, ErrInvalidBody
	}
	now := time.Now()
	return Comment{
		id:        shared.NewID(),
		issueID:   issueID,
		body:      body,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute rebuilds a Comment from persisted data.
func Reconstitute(id, issueID shared.ID, body Body, createdAt, updatedAt time.Time) Comment {
	return Comment{
		id:        id,
		issueID:   issueID,
		body:      body,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (c Comment) ID() shared.ID        { return c.id }
func (c Comment) IssueID() shared.ID   { return c.issueID }
func (c Comment) Body() Body           { return c.body }
func (c Comment) CreatedAt() time.Time { return c.createdAt }
func (c Comment) UpdatedAt() time.Time { return c.updatedAt }
