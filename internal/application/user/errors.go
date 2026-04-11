package user

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already in use")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrTokenNotFound       = errors.New("refresh token not found")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)
