package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewChange(t *testing.T) {
	stage := Stage{ID: uuid.New(), Name: "Crear login", Description: "Detalle"}

	change, err := NewChange(stage, ChangeEdit, " Incongruencia ", " PM ")

	if err != nil {
		t.Fatal(err)
	}
	if change.StageID != stage.ID || change.Name != "Crear login" || change.Description != "Detalle" {
		t.Errorf("cambio = %+v", change)
	}
	if change.ID == uuid.Nil || change.Kind != ChangeEdit || change.Reason != "Incongruencia" || change.RequestedBy != "PM" {
		t.Errorf("cambio = %+v", change)
	}
}

func TestNewChangeRequiresReasonAndRequester(t *testing.T) {
	tests := []struct {
		name        string
		reason      string
		requestedBy string
		want        error
	}{
		{"sin motivo", " ", "PM", ErrChangeReasonRequired},
		{"sin solicitante", "Motivo", " ", ErrChangeRequesterRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewChange(Stage{}, ChangeEdit, tt.reason, tt.requestedBy); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
