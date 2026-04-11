package project_test

import (
	"errors"
	"testing"

	"project-management-tools/internal/domain/project"
	"project-management-tools/internal/domain/shared"
)

func TestNewName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid name", input: "My Project", wantErr: nil},
		{name: "valid name with spaces trimmed", input: "  My Project  ", wantErr: nil},
		{name: "empty string", input: "", wantErr: project.ErrInvalidName},
		{name: "only spaces", input: "   ", wantErr: project.ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := project.NewName(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("valid project is created with active status", func(t *testing.T) {
		name, _ := project.NewName("My Project")
		p, err := project.New(shared.NewID(), name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID().IsZero() {
			t.Error("expected non-zero ID")
		}
		if p.Status() != project.StatusActive {
			t.Errorf("expected status %q, got %q", project.StatusActive, p.Status())
		}
		if p.Name().String() != "My Project" {
			t.Errorf("expected name %q, got %q", "My Project", p.Name().String())
		}
	})
}

func TestProject_Rename(t *testing.T) {
	tests := []struct {
		name    string
		newName string
		wantErr error
	}{
		{name: "valid rename", newName: "New Name", wantErr: nil},
		{name: "empty name rejected", newName: "", wantErr: project.ErrInvalidName},
		{name: "blank name rejected", newName: "  ", wantErr: project.ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, _ := project.NewName("Original")
			p, _ := project.New(shared.NewID(), name)

			newName, _ := project.NewName(tt.newName)
			err := p.Rename(newName)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && p.Name().String() != tt.newName {
				t.Errorf("expected name %q, got %q", tt.newName, p.Name().String())
			}
		})
	}
}

func TestProject_Archive(t *testing.T) {
	t.Run("archiving changes status to archived", func(t *testing.T) {
		name, _ := project.NewName("My Project")
		p, _ := project.New(shared.NewID(), name)

		p.Archive()

		if p.Status() != project.StatusArchived {
			t.Errorf("expected status %q, got %q", project.StatusArchived, p.Status())
		}
	})
}
