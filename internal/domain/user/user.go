package user

import (
	"time"

	"project-management-tools/internal/domain/shared"
)

type User struct {
	id           shared.ID
	email        Email
	passwordHash PasswordHash
	createdAt    time.Time
	updatedAt    time.Time
}

func New(email Email, passwordHash PasswordHash) User {
	now := time.Now()
	return User{
		id:           shared.NewID(),
		email:        email,
		passwordHash: passwordHash,
		createdAt:    now,
		updatedAt:    now,
	}
}

// Reconstitute rebuilds a User from persisted data.
// Bypasses constructor validation — callers must ensure data integrity.
func Reconstitute(id shared.ID, email Email, passwordHash PasswordHash, createdAt, updatedAt time.Time) User {
	return User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (u User) ID() shared.ID               { return u.id }
func (u User) Email() Email                { return u.email }
func (u User) PasswordHash() PasswordHash  { return u.passwordHash }
func (u User) CreatedAt() time.Time        { return u.createdAt }
func (u User) UpdatedAt() time.Time        { return u.updatedAt }
