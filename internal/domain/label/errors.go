package label

import "errors"

var (
	ErrNotFound        = errors.New("label not found")
	ErrInvalidName     = errors.New("label name cannot be empty")
	ErrInvalidColor    = errors.New("label color must be a valid hex color (e.g. #ff0000)")
	ErrDuplicateName   = errors.New("label name already exists in this project")
	ErrInvalidProjectID = errors.New("invalid project id")
)
