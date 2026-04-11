package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"project-management-tools/internal/domain/shared"
	"project-management-tools/internal/domain/user"
)

const refreshTTL = 7 * 24 * time.Hour

type RegisterInput struct {
	Email    string
	Password string
}

type Service struct {
	userRepo  Repository
	tokenRepo TokenRepository
}

func NewService(userRepo Repository, tokenRepo TokenRepository) *Service {
	return &Service{userRepo: userRepo, tokenRepo: tokenRepo}
}

func (s *Service) GetByID(ctx context.Context, id shared.ID) (user.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (user.User, error) {
	email, err := user.NewEmail(input.Email)
	if err != nil {
		return user.User{}, err
	}

	_, err = s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return user.User{}, ErrEmailAlreadyExists
	}
	if !errors.Is(err, user.ErrNotFound) {
		return user.User{}, err
	}

	hash, err := user.NewPasswordHash(input.Password)
	if err != nil {
		return user.User{}, err
	}

	u := user.New(email, hash)
	if err := s.userRepo.Save(ctx, u); err != nil {
		return user.User{}, err
	}

	return u, nil
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (user.User, error) {
	e, err := user.NewEmail(email)
	if err != nil {
		return user.User{}, ErrInvalidCredentials
	}

	u, err := s.userRepo.FindByEmail(ctx, e)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return user.User{}, ErrInvalidCredentials
		}
		return user.User{}, err
	}

	if !u.PasswordHash().Matches(password) {
		return user.User{}, ErrInvalidCredentials
	}

	return u, nil
}

func (s *Service) IssueRefreshToken(ctx context.Context, userID shared.ID) (string, error) {
	token := uuid.New().String()
	rt := RefreshToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(refreshTTL),
		CreatedAt: time.Now(),
	}
	if err := s.tokenRepo.SaveRefreshToken(ctx, rt); err != nil {
		return "", err
	}
	return token, nil
}

// RotateRefreshToken validates the given token, revokes it, and issues a new one.
// Returns the user and the new refresh token.
func (s *Service) RotateRefreshToken(ctx context.Context, token string) (user.User, string, error) {
	rt, err := s.tokenRepo.FindRefreshToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return user.User{}, "", ErrInvalidRefreshToken
		}
		return user.User{}, "", err
	}

	if time.Now().After(rt.ExpiresAt) {
		_ = s.tokenRepo.DeleteRefreshToken(ctx, token)
		return user.User{}, "", ErrRefreshTokenExpired
	}

	u, err := s.userRepo.FindByID(ctx, rt.UserID)
	if err != nil {
		return user.User{}, "", err
	}

	if err := s.tokenRepo.DeleteRefreshToken(ctx, token); err != nil {
		return user.User{}, "", err
	}

	newToken, err := s.IssueRefreshToken(ctx, u.ID())
	if err != nil {
		return user.User{}, "", err
	}

	return u, newToken, nil
}

func (s *Service) RevokeRefreshToken(ctx context.Context, token string) error {
	return s.tokenRepo.DeleteRefreshToken(ctx, token)
}
