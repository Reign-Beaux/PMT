package user_test

import (
	"errors"
	"testing"

	"project-management-tools/internal/domain/user"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
		wantVal string
	}{
		{name: "valid email", input: "user@example.com", wantErr: nil, wantVal: "user@example.com"},
		{name: "uppercase is normalized", input: "User@Example.COM", wantErr: nil, wantVal: "user@example.com"},
		{name: "leading/trailing spaces trimmed", input: "  user@example.com  ", wantErr: nil, wantVal: "user@example.com"},
		{name: "missing @", input: "userexample.com", wantErr: user.ErrInvalidEmail},
		{name: "missing domain dot", input: "user@example", wantErr: user.ErrInvalidEmail},
		{name: "empty string", input: "", wantErr: user.ErrInvalidEmail},
		{name: "dot at end of domain", input: "user@example.", wantErr: user.ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := user.NewEmail(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && e.String() != tt.wantVal {
				t.Errorf("got email %q, want %q", e.String(), tt.wantVal)
			}
		})
	}
}

func TestNewPasswordHash(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		wantErr   error
	}{
		{name: "valid password", plaintext: "secret123", wantErr: nil},
		{name: "exactly 8 characters", plaintext: "12345678", wantErr: nil},
		{name: "too short", plaintext: "short", wantErr: user.ErrInvalidPassword},
		{name: "7 characters", plaintext: "1234567", wantErr: user.ErrInvalidPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := user.NewPasswordHash(tt.plaintext)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if !h.Matches(tt.plaintext) {
					t.Error("hash does not match original plaintext")
				}
				if h.Matches("wrongpassword") {
					t.Error("hash matches wrong password")
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("user is created with correct fields", func(t *testing.T) {
		email, _ := user.NewEmail("alice@example.com")
		hash, _ := user.NewPasswordHash("password123")

		u := user.New(email, hash)

		if u.ID().IsZero() {
			t.Error("expected non-zero ID")
		}
		if u.Email().String() != "alice@example.com" {
			t.Errorf("unexpected email: %q", u.Email().String())
		}
		if !u.PasswordHash().Matches("password123") {
			t.Error("password hash does not match")
		}
	})
}
