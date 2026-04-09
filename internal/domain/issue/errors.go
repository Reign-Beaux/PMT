package issue

import "errors"

var (
	ErrNotFound          = errors.New("issue not found")
	ErrInvalidTitle      = errors.New("issue title cannot be empty")
	ErrInvalidProjectID  = errors.New("invalid project id")
	ErrInvalidTransition = errors.New("invalid status transition")
)
