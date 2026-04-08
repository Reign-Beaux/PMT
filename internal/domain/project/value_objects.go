package project

import (
	"errors"
	"strings"
)

// Name

type Name struct {
	value string
}

func NewName(s string) (Name, error) {
	n := Name{value: strings.TrimSpace(s)}
	if !n.isValid() {
		return Name{}, ErrInvalidName
	}
	return n, nil
}

func (n Name) String() string { return n.value }
func (n Name) isValid() bool  { return n.value != "" }

// Status

type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

var ErrInvalidStatus = errors.New("invalid project status")

func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusActive, StatusArchived:
		return Status(s), nil
	default:
		return "", ErrInvalidStatus
	}
}
