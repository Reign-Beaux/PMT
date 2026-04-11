package user

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Email

type Email struct{ value string }

func NewEmail(s string) (Email, error) {
	e := Email{value: strings.ToLower(strings.TrimSpace(s))}
	if !e.isValid() {
		return Email{}, ErrInvalidEmail
	}
	return e, nil
}

func (e Email) String() string { return e.value }
func (e Email) isValid() bool {
	at := strings.Index(e.value, "@")
	if at < 1 {
		return false
	}
	domain := e.value[at+1:]
	dot := strings.LastIndex(domain, ".")
	return dot >= 1 && dot < len(domain)-1
}

// PasswordHash

type PasswordHash struct{ value string }

// NewPasswordHash hashes plaintext with bcrypt. Enforces minimum length.
func NewPasswordHash(plaintext string) (PasswordHash, error) {
	if len(plaintext) < 8 {
		return PasswordHash{}, ErrInvalidPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return PasswordHash{}, err
	}
	return PasswordHash{value: string(hash)}, nil
}

// ReconstitutePasswordHash rebuilds a PasswordHash from a persisted hash.
// Only use when loading from storage — never for raw passwords.
func ReconstitutePasswordHash(hash string) PasswordHash {
	return PasswordHash{value: hash}
}

func (p PasswordHash) String() string { return p.value }

func (p PasswordHash) Matches(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.value), []byte(plaintext)) == nil
}
