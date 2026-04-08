package phase

import "errors"

var (
	ErrNotFound        = errors.New("phase not found")
	ErrInvalidName     = errors.New("phase name cannot be empty")
	ErrInvalidOrder    = errors.New("phase order must be greater than zero")
	ErrInvalidProjectID = errors.New("invalid project id")
)
