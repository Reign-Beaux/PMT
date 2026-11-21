package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

var (
	// ErrTokenNameInvalid indica que el nombre del token está vacío o es muy largo.
	ErrTokenNameInvalid = kernel.Invalid("el nombre del token debe tener entre 1 y 100 caracteres")
	// ErrTokenNotFound indica que el token no existe, ya está revocado o es de otro usuario.
	ErrTokenNotFound = kernel.NotFound("token no encontrado")
)

// APIToken es un token de API de un usuario. El token en sí solo se muestra al
// crearlo: se guarda su hash.
type APIToken struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// ValidateTokenName quita espacios y exige de 1 a 100 caracteres.
func ValidateTokenName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return "", ErrTokenNameInvalid
	}
	return name, nil
}
