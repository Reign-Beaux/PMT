package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewBug(t *testing.T) {
	projectID := uuid.New()

	b, err := NewBug(projectID, " Falla ", " Detalle ", " Paso 1 ", " 1.2.0 ")

	if err != nil {
		t.Fatal(err)
	}
	if b.Status != StatusReported || b.ProjectID != projectID {
		t.Errorf("bug = %+v", b)
	}
	if b.Title != "Falla" || b.Description != "Detalle" || b.StepsToReproduce != "Paso 1" || b.ProductVersion != "1.2.0" {
		t.Errorf("bug = %+v", b)
	}
}

func TestNewBugValidates(t *testing.T) {
	tests := []struct {
		name, title, steps, version string
		want                        error
	}{
		{"título", " ", "P", "1", ErrBugTitleRequired},
		{"pasos", "T", "", "1", ErrBugStepsRequired},
		{"versión", "T", "P", " ", ErrBugVersionRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewBug(uuid.New(), tt.title, "", tt.steps, tt.version); !errors.Is(err, tt.want) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestUpdateKeepsBugWhenInvalid(t *testing.T) {
	b, err := NewBug(uuid.New(), "T", "D", "P", "1")
	if err != nil {
		t.Fatal(err)
	}

	if err := b.Update("Otro", "", "", "2"); !errors.Is(err, ErrBugStepsRequired) {
		t.Fatalf("err = %v", err)
	}
	if b.Title != "T" || b.ProductVersion != "1" {
		t.Errorf("bug = %+v", b)
	}
}
