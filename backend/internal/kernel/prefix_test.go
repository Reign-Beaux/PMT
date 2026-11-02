package kernel

import (
	"errors"
	"testing"
)

func TestPrefixKeepsKindAndCause(t *testing.T) {
	cause := Conflict("no")

	err := Prefix("tarea 1: ", cause)

	var business *Error
	if !errors.As(err, &business) || business.Kind != KindConflict || err.Error() != "tarea 1: no" {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, cause) {
		t.Error("debe conservar el error original")
	}
}

func TestPrefixWrapsOtherErrors(t *testing.T) {
	cause := errors.New("fallo")

	err := Prefix("tarea 1: ", cause)

	var business *Error
	if errors.As(err, &business) || !errors.Is(err, cause) {
		t.Fatalf("err = %v", err)
	}
}
