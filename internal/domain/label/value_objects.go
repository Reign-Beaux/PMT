package label

import (
	"regexp"
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

// Color

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type Color struct {
	value string
}

// NewColor validates and constructs a Color from a hex string (e.g. "#ff0000").
// The default color "#6366f1" is used when the input is empty.
func NewColor(s string) (Color, error) {
	if s == "" {
		return Color{value: "#6366f1"}, nil
	}
	if !hexColorRe.MatchString(s) {
		return Color{}, ErrInvalidColor
	}
	return Color{value: strings.ToLower(s)}, nil
}

func (c Color) String() string { return c.value }
