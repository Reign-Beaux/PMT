package user

import (
	"context"
	"time"

	"project-management-tools/internal/domain/shared"
	"project-management-tools/internal/domain/user"
)

// Repository is the driven port for user persistence.
// Defined here, in the consumer (application layer).
type Repository interface {
	Save(ctx context.Context, u user.User) error
	FindByID(ctx context.Context, id shared.ID) (user.User, error)
	FindByEmail(ctx context.Context, email user.Email) (user.User, error)
}

// TokenRepository is the driven port for refresh token persistence.
type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, t RefreshToken) error
	FindRefreshToken(ctx context.Context, token string) (RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteRefreshTokensByUserID(ctx context.Context, userID shared.ID) error
}

// RefreshToken is an application-level value object representing a stored refresh token.
type RefreshToken struct {
	Token     string
	UserID    shared.ID
	ExpiresAt time.Time
	CreatedAt time.Time
}
