package phase_test

import (
	"errors"
	"testing"

	"project-management-tools/internal/domain/phase"
	"project-management-tools/internal/domain/shared"
)

func TestNewName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid name", input: "Backend", wantErr: nil},
		{name: "trimmed spaces", input: "  Backend  ", wantErr: nil},
		{name: "empty string", input: "", wantErr: phase.ErrInvalidName},
		{name: "only spaces", input: "   ", wantErr: phase.ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := phase.NewName(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewOrder(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr error
	}{
		{name: "valid order 1", input: 1, wantErr: nil},
		{name: "valid order 10", input: 10, wantErr: nil},
		{name: "zero is invalid", input: 0, wantErr: phase.ErrInvalidOrder},
		{name: "negative is invalid", input: -1, wantErr: phase.ErrInvalidOrder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := phase.NewOrder(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	validProjectID := shared.NewID()
	validName, _ := phase.NewName("Backend")
	validOrder, _ := phase.NewOrder(1)

	tests := []struct {
		name      string
		projectID shared.ID
		phaseName phase.Name
		order     phase.Order
		wantErr   error
	}{
		{
			name:      "valid phase",
			projectID: validProjectID,
			phaseName: validName,
			order:     validOrder,
			wantErr:   nil,
		},
		{
			name:      "zero project id rejected",
			projectID: shared.ID{},
			phaseName: validName,
			order:     validOrder,
			wantErr:   phase.ErrInvalidProjectID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := phase.New(tt.projectID, tt.phaseName, tt.order)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if p.ID().IsZero() {
					t.Error("expected non-zero ID")
				}
				if p.Status() != phase.StatusActive {
					t.Errorf("expected status %q, got %q", phase.StatusActive, p.Status())
				}
			}
		})
	}
}

func TestPhase_Complete(t *testing.T) {
	t.Run("completing changes status to completed", func(t *testing.T) {
		projectID := shared.NewID()
		name, _ := phase.NewName("Backend")
		order, _ := phase.NewOrder(1)
		p, _ := phase.New(projectID, name, order)

		p.Complete()

		if p.Status() != phase.StatusCompleted {
			t.Errorf("expected status %q, got %q", phase.StatusCompleted, p.Status())
		}
	})
}
