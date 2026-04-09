package comment

import "strings"

type Body struct {
	value string
}

func NewBody(s string) (Body, error) {
	b := Body{value: strings.TrimSpace(s)}
	if !b.isValid() {
		return Body{}, ErrInvalidBody
	}
	return b, nil
}

func (b Body) String() string { return b.value }
func (b Body) isValid() bool  { return b.value != "" }
