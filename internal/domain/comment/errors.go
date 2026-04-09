package comment

import "errors"

var (
	ErrNotFound    = errors.New("comment not found")
	ErrInvalidBody = errors.New("comment body cannot be empty")
)
