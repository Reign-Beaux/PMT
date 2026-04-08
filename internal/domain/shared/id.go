package shared

import (
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidID = errors.New("invalid id")

type ID struct {
	value uuid.UUID
}

func NewID() ID {
	return ID{value: uuid.New()}
}

func ParseID(s string) (ID, error) {
	v, err := uuid.Parse(s)
	if err != nil {
		return ID{}, ErrInvalidID
	}
	return ID{value: v}, nil
}

func (id ID) String() string {
	return id.value.String()
}

func (id ID) IsZero() bool {
	return id.value == uuid.Nil
}
