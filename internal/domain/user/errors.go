package user

import "errors"

var (
	ErrNotFound        = errors.New("user not found")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("password must be at least 8 characters")
)
