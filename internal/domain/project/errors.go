package project

import "errors"

var (
	ErrNotFound    = errors.New("project not found")
	ErrInvalidName = errors.New("project name cannot be empty")
)
