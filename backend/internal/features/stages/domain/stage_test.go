package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewStage(t *testing.T) {
	projectID := uuid.New()

	s, err := NewStage(projectID, "  Diseño  ", "  Inicial  ")

	if err != nil {
		t.Fatal(err)
	}
	if s.ID == uuid.Nil || s.ProjectID != projectID {
		t.Errorf("etapa = %+v", s)
	}
	if s.Name != "Diseño" || s.Description != "Inicial" {
		t.Errorf("etapa = %+v", s)
	}
}

func TestNewStageRequiresName(t *testing.T) {
	if _, err := NewStage(uuid.New(), " ", ""); !errors.Is(err, ErrStageNameRequired) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateKeepsDataWhenInvalid(t *testing.T) {
	s, err := NewStage(uuid.New(), "Diseño", "Inicial")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Update("", "Otro"); !errors.Is(err, ErrStageNameRequired) {
		t.Fatalf("err = %v", err)
	}
	if s.Name != "Diseño" || s.Description != "Inicial" {
		t.Errorf("etapa = %+v", s)
	}
}
