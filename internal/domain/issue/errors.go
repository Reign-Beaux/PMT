package issue

import "errors"

var (
	ErrNotFound      = errors.New("issue not found")
	ErrInvalidTitle  = errors.New("issue title cannot be empty")
	ErrInvalidPhaseID = errors.New("invalid phase id")
	ErrInvalidTransition = errors.New("invalid status transition")
)
