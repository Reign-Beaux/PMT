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

func TestNewOwnPlanningStage(t *testing.T) {
	s, err := NewOwnPlanningStage(uuid.New(), " Pagos ", "", " Petición ", " PM ")

	if err != nil {
		t.Fatal(err)
	}
	if !s.InOwnPlanning() || s.AddReason != "Petición" || s.AddRequestedBy != "PM" || s.Status != StatusPlanning {
		t.Errorf("etapa = %+v", s)
	}
	if !errors.Is(s.EnsureHistoryRule(), ErrStageInOwnPlanning) {
		t.Errorf("err = %v", s.EnsureHistoryRule())
	}
}

func TestNewOwnPlanningStageValidates(t *testing.T) {
	tests := []struct {
		name, stage, reason, requestedBy string
		want                             error
	}{
		{"nombre", "", "M", "PM", ErrStageNameRequired},
		{"motivo", "A", " ", "PM", ErrChangeReasonRequired},
		{"solicitante", "A", "M", "", ErrChangeRequesterRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOwnPlanningStage(uuid.New(), tt.stage, "", tt.reason, tt.requestedBy)
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestStageDefinedInPlanningIsNotOwnPlanning(t *testing.T) {
	s, err := NewStage(uuid.New(), "A", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.InOwnPlanning() || s.EnsureHistoryRule() != nil {
		t.Errorf("etapa = %+v", s)
	}
}
