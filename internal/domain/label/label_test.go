package label_test

import (
	"errors"
	"testing"

	"project-management-tools/internal/domain/label"
	"project-management-tools/internal/domain/shared"
)

func TestNewName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr error
	}{
		{input: "bug", wantErr: nil},
		{input: "  feature  ", wantErr: nil},
		{input: "", wantErr: label.ErrInvalidName},
		{input: "   ", wantErr: label.ErrInvalidName},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := label.NewName(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewColor(t *testing.T) {
	tests := []struct {
		input   string
		wantErr error
	}{
		{input: "#ff0000", wantErr: nil},
		{input: "#FF0000", wantErr: nil},
		{input: "#6366f1", wantErr: nil},
		{input: "", wantErr: nil}, // uses default color
		{input: "red", wantErr: label.ErrInvalidColor},
		{input: "#fff", wantErr: label.ErrInvalidColor},
		{input: "#gggggg", wantErr: label.ErrInvalidColor},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := label.NewColor(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	projectID := shared.NewID()
	name, _ := label.NewName("bug")
	color, _ := label.NewColor("#ff0000")

	iss, err := label.New(projectID, name, color)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iss.ID().IsZero() {
		t.Error("expected non-zero ID")
	}
	if iss.Name().String() != "bug" {
		t.Errorf("expected name %q, got %q", "bug", iss.Name().String())
	}
	if iss.Color().String() != "#ff0000" {
		t.Errorf("expected color %q, got %q", "#ff0000", iss.Color().String())
	}
}

func TestLabel_Rename(t *testing.T) {
	projectID := shared.NewID()
	name, _ := label.NewName("bug")
	color, _ := label.NewColor("")
	l, _ := label.New(projectID, name, color)

	newName, _ := label.NewName("feature")
	if err := l.Rename(newName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Name().String() != "feature" {
		t.Errorf("expected name %q, got %q", "feature", l.Name().String())
	}
}
