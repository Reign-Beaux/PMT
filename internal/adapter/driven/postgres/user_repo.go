package postgres

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	userapp "project-management-tools/internal/application/user"
	"project-management-tools/internal/domain/shared"
	"project-management-tools/internal/domain/user"
)

// ── User ─────────────────────────────────────────────────────────────────────

type userModel struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Email        string    `gorm:"not null;uniqueIndex"`
	PasswordHash string    `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userModel) TableName() string { return "users" }

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, u user.User) error {
	return r.db.WithContext(ctx).Create(&userModel{
		ID:           u.ID().String(),
		Email:        u.Email().String(),
		PasswordHash: u.PasswordHash().String(),
		CreatedAt:    u.CreatedAt(),
		UpdatedAt:    u.UpdatedAt(),
	}).Error
}

func (r *UserRepository) FindByID(ctx context.Context, id shared.ID) (user.User, error) {
	var m userModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, err
	}
	return toUserDomain(m)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email user.Email) (user.User, error) {
	var m userModel
	err := r.db.WithContext(ctx).First(&m, "email = ?", email.String()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, err
	}
	return toUserDomain(m)
}

func toUserDomain(m userModel) (user.User, error) {
	id, err := shared.ParseID(m.ID)
	if err != nil {
		return user.User{}, err
	}
	email, err := user.NewEmail(m.Email)
	if err != nil {
		return user.User{}, err
	}
	hash := user.ReconstitutePasswordHash(m.PasswordHash)
	return user.Reconstitute(id, email, hash, m.CreatedAt, m.UpdatedAt), nil
}

// ── Refresh tokens ────────────────────────────────────────────────────────────

type refreshTokenModel struct {
	Token     string    `gorm:"primaryKey"`
	UserID    string    `gorm:"not null;index"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (refreshTokenModel) TableName() string { return "refresh_tokens" }

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) SaveRefreshToken(ctx context.Context, t userapp.RefreshToken) error {
	return r.db.WithContext(ctx).Create(&refreshTokenModel{
		Token:     t.Token,
		UserID:    t.UserID.String(),
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}).Error
}

func (r *TokenRepository) FindRefreshToken(ctx context.Context, token string) (userapp.RefreshToken, error) {
	var m refreshTokenModel
	err := r.db.WithContext(ctx).First(&m, "token = ?", token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return userapp.RefreshToken{}, userapp.ErrTokenNotFound
	}
	if err != nil {
		return userapp.RefreshToken{}, err
	}

	userID, err := shared.ParseID(m.UserID)
	if err != nil {
		return userapp.RefreshToken{}, err
	}

	return userapp.RefreshToken{
		Token:     m.Token,
		UserID:    userID,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}, nil
}

func (r *TokenRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Delete(&refreshTokenModel{}, "token = ?", token).Error
}

func (r *TokenRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID shared.ID) error {
	return r.db.WithContext(ctx).Delete(&refreshTokenModel{}, "user_id = ?", userID.String()).Error
}
