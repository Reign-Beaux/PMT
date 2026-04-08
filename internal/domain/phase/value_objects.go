package phase

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

// Order

type Order struct {
	value int
}

func NewOrder(v int) (Order, error) {
	o := Order{value: v}
	if !o.isValid() {
		return Order{}, ErrInvalidOrder
	}
	return o, nil
}

func (o Order) Value() int    { return o.value }
func (o Order) isValid() bool { return o.value > 0 }

// Status

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

var ErrInvalidStatus = errors.New("invalid phase status")

func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusActive, StatusCompleted:
		return Status(s), nil
	default:
		return "", ErrInvalidStatus
	}
}
