package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewProject(t *testing.T) {
	p, err := NewProject("  PMT  ", "  Gestor  ")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == uuid.Nil {
		t.Error("ID vacío")
	}
	if p.Name != "PMT" || p.Description != "Gestor" {
		t.Errorf("proyecto = %+v", p)
	}
}

func TestNewProjectRequiresName(t *testing.T) {
	if _, err := NewProject("   ", ""); !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateKeepsDataWhenInvalid(t *testing.T) {
	p, err := NewProject("PMT", "Gestor")
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Update("", "Otro"); !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if p.Name != "PMT" || p.Description != "Gestor" {
		t.Errorf("proyecto = %+v", p)
	}
}
