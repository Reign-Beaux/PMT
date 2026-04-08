package issue

import (
	"errors"
	"strings"
)

// Title

type Title struct {
	value string
}

func NewTitle(s string) (Title, error) {
	t := Title{value: strings.TrimSpace(s)}
	if !t.isValid() {
		return Title{}, ErrInvalidTitle
	}
	return t, nil
}

func (t Title) String() string { return t.value }
func (t Title) isValid() bool  { return t.value != "" }

// Status

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusClosed     Status = "closed"
)

var ErrInvalidStatus = errors.New("invalid issue status")

func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusOpen, StatusInProgress, StatusDone, StatusClosed:
		return Status(s), nil
	default:
		return "", ErrInvalidStatus
	}
}

// Priority

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

var ErrInvalidPriority = errors.New("invalid issue priority")

func ParsePriority(s string) (Priority, error) {
	switch Priority(s) {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return Priority(s), nil
	default:
		return "", ErrInvalidPriority
	}
}
